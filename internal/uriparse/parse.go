package uriparse

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type ParsedOutbound struct {
	Tag      string
	Type     string
	Outbound map[string]any
}

func Parse(text string) ([]ParsedOutbound, error) {
	lines := strings.Split(text, "\n")
	outbounds := make([]ParsedOutbound, 0, len(lines))
	for i, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		outbound, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		outbounds = append(outbounds, outbound)
	}
	if len(outbounds) == 0 {
		return nil, errors.New("no URI lines found")
	}

	seen := map[string]int{}
	for i := range outbounds {
		tag := uniqueTag(outbounds[i].Tag, seen)
		outbounds[i].Tag = tag
		outbounds[i].Outbound["tag"] = tag
	}
	return outbounds, nil
}

func parseLine(line string) (ParsedOutbound, error) {
	switch {
	case strings.HasPrefix(line, "ss://"):
		return parseShadowsocks(line)
	case strings.HasPrefix(line, "vmess://"):
		return parseVMess(line)
	case strings.HasPrefix(line, "vless://"):
		return parseVLESS(line)
	case strings.HasPrefix(line, "trojan://"):
		return parseTrojan(line)
	case strings.HasPrefix(line, "hysteria2://"):
		return parseHysteria2(line)
	default:
		return ParsedOutbound{}, errors.New("unsupported URI scheme")
	}
}

func parseShadowsocks(line string) (ParsedOutbound, error) {
	withoutScheme := strings.TrimPrefix(line, "ss://")
	main, fragment := splitFragment(withoutScheme)
	main = strings.SplitN(main, "?", 2)[0]

	var method, password, host string
	var port uint16
	if strings.Contains(main, "@") {
		parsed, err := url.Parse("ss://" + withoutScheme)
		if err != nil {
			return ParsedOutbound{}, err
		}
		host, port, err = splitHostPort(parsed.Host)
		if err != nil {
			return ParsedOutbound{}, err
		}
		method, password, err = decodeSSUserInfo(parsed.User)
		if err != nil {
			return ParsedOutbound{}, err
		}
		if parsed.Fragment != "" {
			fragment = parsed.Fragment
		}
	} else {
		decoded, err := decodeBase64Flexible(main)
		if err != nil {
			return ParsedOutbound{}, fmt.Errorf("decode shadowsocks payload: %w", err)
		}
		userInfo, hostPort, ok := strings.Cut(decoded, "@")
		if !ok {
			return ParsedOutbound{}, errors.New("missing shadowsocks host")
		}
		method, password, ok = strings.Cut(userInfo, ":")
		if !ok {
			return ParsedOutbound{}, errors.New("missing shadowsocks method or password")
		}
		host, port, err = splitHostPort(hostPort)
		if err != nil {
			return ParsedOutbound{}, err
		}
	}
	if method == "" || password == "" {
		return ParsedOutbound{}, errors.New("missing shadowsocks method or password")
	}
	tag := firstNonEmpty(fragment, generatedTag("ss", host, port))
	out := baseOutbound("shadowsocks", tag, host, port)
	out["method"] = method
	out["password"] = password
	return parsed(out), nil
}

func decodeSSUserInfo(user *url.Userinfo) (string, string, error) {
	if user == nil {
		return "", "", errors.New("missing shadowsocks user info")
	}
	username := user.Username()
	if password, ok := user.Password(); ok {
		return username, password, nil
	}
	decoded, err := decodeBase64Flexible(username)
	if err != nil {
		return "", "", err
	}
	method, password, ok := strings.Cut(decoded, ":")
	if !ok {
		return "", "", errors.New("missing shadowsocks method or password")
	}
	return method, password, nil
}

func parseVMess(line string) (ParsedOutbound, error) {
	encoded := strings.TrimPrefix(line, "vmess://")
	decoded, err := decodeBase64Flexible(encoded)
	if err != nil {
		return ParsedOutbound{}, fmt.Errorf("decode vmess payload: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(decoded), &raw); err != nil {
		return ParsedOutbound{}, fmt.Errorf("decode vmess json: %w", err)
	}
	host := stringValue(raw["add"])
	port, err := portValue(raw["port"])
	if err != nil {
		return ParsedOutbound{}, err
	}
	uuid := stringValue(raw["id"])
	if host == "" || uuid == "" {
		return ParsedOutbound{}, errors.New("missing vmess host or uuid")
	}
	tag := firstNonEmpty(stringValue(raw["ps"]), generatedTag("vmess", host, port))
	out := baseOutbound("vmess", tag, host, port)
	out["uuid"] = uuid
	out["security"] = firstNonEmpty(stringValue(raw["scy"]), "auto")
	if alterID, err := intValue(raw["aid"]); err == nil && alterID > 0 {
		out["alter_id"] = alterID
	}
	network := stringValue(raw["net"])
	if err := applyV2RayTransport(out, network, stringValue(raw["path"]), stringValue(raw["host"])); err != nil {
		return ParsedOutbound{}, err
	}
	if stringValue(raw["tls"]) == "tls" {
		applyTLS(out, host, stringValue(raw["sni"]), false, nil)
	}
	return parsed(out), nil
}

func parseVLESS(line string) (ParsedOutbound, error) {
	parsedURL, err := url.Parse(line)
	if err != nil {
		return ParsedOutbound{}, err
	}
	host, port, err := splitHostPort(parsedURL.Host)
	if err != nil {
		return ParsedOutbound{}, err
	}
	uuid := parsedURL.User.Username()
	if uuid == "" {
		return ParsedOutbound{}, errors.New("missing vless uuid")
	}
	query := parsedURL.Query()
	if encryption := query.Get("encryption"); encryption != "" && encryption != "none" {
		return ParsedOutbound{}, errors.New("unsupported vless encryption")
	}
	tag := firstNonEmpty(parsedURL.Fragment, generatedTag("vless", host, port))
	out := baseOutbound("vless", tag, host, port)
	out["uuid"] = uuid
	if flow := query.Get("flow"); flow != "" {
		out["flow"] = flow
	}
	if err := applyV2RayTransport(out, query.Get("type"), query.Get("path"), query.Get("host")); err != nil {
		return ParsedOutbound{}, err
	}
	applyQueryTLS(out, host, query, false)
	return parsed(out), nil
}

func parseTrojan(line string) (ParsedOutbound, error) {
	parsedURL, err := url.Parse(line)
	if err != nil {
		return ParsedOutbound{}, err
	}
	host, port, err := splitHostPort(parsedURL.Host)
	if err != nil {
		return ParsedOutbound{}, err
	}
	password := parsedURL.User.Username()
	if password == "" {
		return ParsedOutbound{}, errors.New("missing trojan password")
	}
	query := parsedURL.Query()
	tag := firstNonEmpty(parsedURL.Fragment, generatedTag("trojan", host, port))
	out := baseOutbound("trojan", tag, host, port)
	out["password"] = password
	if err := applyV2RayTransport(out, query.Get("type"), query.Get("path"), query.Get("host")); err != nil {
		return ParsedOutbound{}, err
	}
	applyQueryTLS(out, host, query, true)
	return parsed(out), nil
}

func parseHysteria2(line string) (ParsedOutbound, error) {
	parsedURL, err := url.Parse(line)
	if err != nil {
		return ParsedOutbound{}, err
	}
	host, port, err := splitHostPort(parsedURL.Host)
	if err != nil {
		return ParsedOutbound{}, err
	}
	password := parsedURL.User.Username()
	query := parsedURL.Query()
	tag := firstNonEmpty(parsedURL.Fragment, generatedTag("hysteria2", host, port))
	out := baseOutbound("hysteria2", tag, host, port)
	if password != "" {
		out["password"] = password
	}
	if obfs := query.Get("obfs"); obfs != "" {
		out["obfs"] = map[string]any{
			"type":     obfs,
			"password": query.Get("obfs-password"),
		}
	}
	applyQueryTLS(out, host, query, true)
	return parsed(out), nil
}

func parsed(outbound map[string]any) ParsedOutbound {
	return ParsedOutbound{
		Tag:      outbound["tag"].(string),
		Type:     outbound["type"].(string),
		Outbound: outbound,
	}
}

func baseOutbound(typ, tag, host string, port uint16) map[string]any {
	return map[string]any{
		"type":        typ,
		"tag":         tag,
		"server":      host,
		"server_port": int(port),
	}
}

func applyQueryTLS(out map[string]any, host string, query url.Values, defaultEnabled bool) {
	if query.Get("security") == "none" {
		return
	}
	enabled := defaultEnabled || query.Get("security") == "tls" || query.Get("tls") == "1" || query.Get("sni") != ""
	if enabled {
		applyTLS(out, host, firstNonEmpty(query.Get("sni"), query.Get("servername")), boolValue(query.Get("allowInsecure")) || boolValue(query.Get("insecure")), query["alpn"])
	}
}

func applyTLS(out map[string]any, host string, serverName string, insecure bool, alpn []string) {
	tls := map[string]any{"enabled": true}
	if serverName == "" {
		serverName = host
	}
	if serverName != "" {
		tls["server_name"] = serverName
	}
	if insecure {
		tls["insecure"] = true
	}
	if len(alpn) > 0 {
		var values []string
		for _, item := range alpn {
			for _, part := range strings.Split(item, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					values = append(values, part)
				}
			}
		}
		if len(values) > 0 {
			tls["alpn"] = values
		}
	}
	out["tls"] = tls
}

func applyV2RayTransport(out map[string]any, network, path, host string) error {
	switch network {
	case "", "tcp":
		return nil
	case "ws":
		transport := map[string]any{"type": "ws"}
		if path != "" {
			transport["path"] = path
		}
		if host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		out["transport"] = transport
		return nil
	default:
		return fmt.Errorf("unsupported v2ray transport: %s", network)
	}
}

func splitFragment(value string) (string, string) {
	main, fragment, ok := strings.Cut(value, "#")
	if !ok {
		return value, ""
	}
	decoded, err := url.QueryUnescape(fragment)
	if err != nil {
		return main, fragment
	}
	return main, decoded
}

func splitHostPort(value string) (string, uint16, error) {
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		idx := strings.LastIndex(value, ":")
		if idx < 0 {
			return "", 0, errors.New("missing server port")
		}
		host = value[:idx]
		portText = value[idx+1:]
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return "", 0, errors.New("missing server host")
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return "", 0, errors.New("invalid server port")
	}
	return host, uint16(port), nil
}

func decodeBase64Flexible(value string) (string, error) {
	value = strings.TrimSpace(value)
	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return string(decoded), nil
		}
	}
	return "", errors.New("invalid base64")
}

func uniqueTag(tag string, seen map[string]int) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = "outbound"
	}
	count := seen[tag]
	if count == 0 {
		seen[tag] = 1
		return tag
	}
	for {
		count++
		candidate := fmt.Sprintf("%s-%d", tag, count)
		if seen[candidate] == 0 {
			seen[tag] = count
			seen[candidate] = 1
			return candidate
		}
	}
}

func generatedTag(prefix, host string, port uint16) string {
	return fmt.Sprintf("%s-%s-%d", prefix, strings.ReplaceAll(host, ":", "-"), port)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}

func intValue(value any) (int, error) {
	switch v := value.(type) {
	case string:
		return strconv.Atoi(v)
	case float64:
		return int(v), nil
	default:
		return 0, errors.New("not an integer")
	}
}

func portValue(value any) (uint16, error) {
	port, err := intValue(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0, errors.New("invalid server port")
	}
	return uint16(port), nil
}

func boolValue(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
