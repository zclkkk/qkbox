//go:build linux

package capability

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type linuxSystemProxy struct {
	availability SystemProxyAvailability
	desktop      string
	kdeWriteTool string
	kdeReadTool  string
}

func NewSystemProxyProvider() SystemProxyProvider {
	p := &linuxSystemProxy{}
	p.detect()
	return p
}

func (p *linuxSystemProxy) Availability() SystemProxyAvailability {
	return p.availability
}

func (p *linuxSystemProxy) detect() {
	desktop := linuxDesktopName()
	if strings.Contains(desktop, "gnome") {
		if commandExists("gsettings") {
			p.desktop = "gnome"
			p.availability = SystemProxyAvailability{Available: true, Supported: true}
			return
		}
		p.availability = SystemProxyAvailability{Available: true, Supported: false, Reason: "GNOME desktop detected but gsettings is not available."}
		return
	}
	if strings.Contains(desktop, "kde") || strings.Contains(desktop, "plasma") {
		writeTool := kdeWriteTool()
		readTool := kdeReadTool(writeTool)
		if writeTool != "" && readTool != "" {
			p.desktop = "kde"
			p.kdeWriteTool = writeTool
			p.kdeReadTool = readTool
			p.availability = SystemProxyAvailability{Available: true, Supported: true}
			return
		}
		p.availability = SystemProxyAvailability{Available: true, Supported: false, Reason: "KDE desktop detected but kwriteconfig and kreadconfig are not both available."}
		return
	}
	p.availability = SystemProxyAvailability{
		Available: true,
		Supported: false,
		Reason:    "No supported desktop environment detected (GNOME with gsettings or KDE with kwriteconfig required).",
	}
}

type linuxSnapshot struct {
	DE             string      `json:"de"`
	GNOMEMode      string      `json:"gnome_mode,omitempty"`
	GNOMEHTTPHost  string      `json:"gnome_http_host,omitempty"`
	GNOMEHTTPPort  int         `json:"gnome_http_port,omitempty"`
	GNOMEHTTPSHost string      `json:"gnome_https_host,omitempty"`
	GNOMEHTTPSPort int         `json:"gnome_https_port,omitempty"`
	KDEProxyType   configValue `json:"kde_proxy_type,omitempty"`
	KDEHTTPProxy   configValue `json:"kde_http_proxy,omitempty"`
	KDEHTTPSProxy  configValue `json:"kde_https_proxy,omitempty"`
}

type configValue struct {
	Exists bool   `json:"exists"`
	Value  string `json:"value,omitempty"`
}

func (p *linuxSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	switch p.desktop {
	case "gnome":
		return snapshotGNOME()
	case "kde":
		return snapshotKDE()
	}
	return nil, fmt.Errorf("no supported desktop environment")
}

func (p *linuxSystemProxy) Apply(addr string, port int) error {
	switch p.desktop {
	case "gnome":
		return applyGNOME(addr, port)
	case "kde":
		return applyKDE(p.kdeWriteTool, addr, port)
	}
	return fmt.Errorf("no supported desktop environment")
}

func (p *linuxSystemProxy) Restore(snapshot *SystemProxySnapshot) error {
	var snap linuxSnapshot
	if err := json.Unmarshal(snapshot.Raw, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}
	switch snap.DE {
	case "gnome":
		return restoreGNOME(snap)
	case "kde":
		return restoreKDE(p.kdeWriteTool, snap)
	}
	return fmt.Errorf("unknown DE in snapshot: %s", snap.DE)
}

func (p *linuxSystemProxy) CurrentState() (SystemProxyCurrentState, error) {
	switch p.desktop {
	case "gnome":
		return currentStateGNOME()
	case "kde":
		return currentStateKDE(p.kdeReadTool)
	}
	return SystemProxyCurrentState{}, nil
}

func linuxDesktopName() string {
	values := []string{
		os.Getenv("XDG_CURRENT_DESKTOP"),
		os.Getenv("DESKTOP_SESSION"),
		os.Getenv("GDMSESSION"),
	}
	return strings.ToLower(strings.Join(values, " "))
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GNOME

func gsettingsGet(schema, key string) (string, error) {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return "", fmt.Errorf("gsettings get %s %s: %w", schema, key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gsettingsSet(schema, key, value string) error {
	out, err := exec.Command("gsettings", "set", schema, key, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gsettings set %s %s %s: %s: %w", schema, key, value, string(out), err)
	}
	return nil
}

func snapshotGNOME() (*SystemProxySnapshot, error) {
	mode, err := gsettingsGet("org.gnome.system.proxy", "mode")
	if err != nil {
		return nil, err
	}
	httpHost, err := gsettingsGet("org.gnome.system.proxy.http", "host")
	if err != nil {
		return nil, err
	}
	httpPort, err := gsettingsGet("org.gnome.system.proxy.http", "port")
	if err != nil {
		return nil, err
	}
	httpsHost, err := gsettingsGet("org.gnome.system.proxy.https", "host")
	if err != nil {
		return nil, err
	}
	httpsPort, err := gsettingsGet("org.gnome.system.proxy.https", "port")
	if err != nil {
		return nil, err
	}

	httpPortInt, err := strconv.Atoi(httpPort)
	if err != nil {
		return nil, fmt.Errorf("parse GNOME HTTP proxy port: %w", err)
	}
	httpsPortInt, err := strconv.Atoi(httpsPort)
	if err != nil {
		return nil, fmt.Errorf("parse GNOME HTTPS proxy port: %w", err)
	}

	snap := linuxSnapshot{
		DE:             "gnome",
		GNOMEMode:      gsettingsStringValue(mode),
		GNOMEHTTPHost:  gsettingsStringValue(httpHost),
		GNOMEHTTPPort:  httpPortInt,
		GNOMEHTTPSHost: gsettingsStringValue(httpsHost),
		GNOMEHTTPSPort: httpsPortInt,
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return &SystemProxySnapshot{Raw: raw}, nil
}

func applyGNOME(addr string, port int) error {
	portStr := strconv.Itoa(port)
	if err := gsettingsSet("org.gnome.system.proxy.http", "host", gsettingsStringLiteral(addr)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "port", portStr); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "host", gsettingsStringLiteral(addr)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "port", portStr); err != nil {
		return err
	}
	return gsettingsSet("org.gnome.system.proxy", "mode", gsettingsStringLiteral("manual"))
}

func restoreGNOME(snap linuxSnapshot) error {
	if err := gsettingsSet("org.gnome.system.proxy.http", "host", gsettingsStringLiteral(snap.GNOMEHTTPHost)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "port", strconv.Itoa(snap.GNOMEHTTPPort)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "host", gsettingsStringLiteral(snap.GNOMEHTTPSHost)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "port", strconv.Itoa(snap.GNOMEHTTPSPort)); err != nil {
		return err
	}
	return gsettingsSet("org.gnome.system.proxy", "mode", gsettingsStringLiteral(snap.GNOMEMode))
}

func currentStateGNOME() (SystemProxyCurrentState, error) {
	mode, err := gsettingsGet("org.gnome.system.proxy", "mode")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpHost, err := gsettingsGet("org.gnome.system.proxy.http", "host")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpPortStr, err := gsettingsGet("org.gnome.system.proxy.http", "port")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpsHost, err := gsettingsGet("org.gnome.system.proxy.https", "host")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpsPortStr, err := gsettingsGet("org.gnome.system.proxy.https", "port")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpPort, err := strconv.Atoi(httpPortStr)
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpsPort, err := strconv.Atoi(httpsPortStr)
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	http := gsettingsStringValue(httpHost)
	https := gsettingsStringValue(httpsHost)
	enabled := gsettingsStringValue(mode) == "manual" && http == https && httpPort == httpsPort
	return SystemProxyCurrentState{Enabled: enabled, Addr: http, Port: httpPort}, nil
}

func gsettingsStringLiteral(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}

func gsettingsStringValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		value = strings.TrimPrefix(strings.TrimSuffix(value, "'"), "'")
	}
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

// KDE

func kdeWriteTool() string {
	if commandExists("kwriteconfig6") {
		return "kwriteconfig6"
	}
	if commandExists("kwriteconfig5") {
		return "kwriteconfig5"
	}
	return ""
}

func kdeReadTool(writeTool string) string {
	if writeTool == "kwriteconfig6" && commandExists("kreadconfig6") {
		return "kreadconfig6"
	}
	if writeTool == "kwriteconfig5" && commandExists("kreadconfig5") {
		return "kreadconfig5"
	}
	if commandExists("kreadconfig6") {
		return "kreadconfig6"
	}
	if commandExists("kreadconfig5") {
		return "kreadconfig5"
	}
	return ""
}

func snapshotKDE() (*SystemProxySnapshot, error) {
	path, err := kdeConfigPath()
	if err != nil {
		return nil, err
	}
	snap := linuxSnapshot{
		DE:            "kde",
		KDEProxyType:  readKDEConfigValue(path, "Proxy Settings", "ProxyType"),
		KDEHTTPProxy:  readKDEConfigValue(path, "Proxy Settings", "httpProxy"),
		KDEHTTPSProxy: readKDEConfigValue(path, "Proxy Settings", "httpsProxy"),
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return &SystemProxySnapshot{Raw: raw}, nil
}

func applyKDE(tool, addr string, port int) error {
	portStr := strconv.Itoa(port)
	group := "Proxy Settings"
	file := "kioslaverc"

	for _, key := range []string{"httpProxy", "httpsProxy"} {
		out, err := exec.Command(tool, "--file", file, "--group", group, "--key", key, "http://"+addr+":"+portStr).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %s: %s: %w", tool, key, string(out), err)
		}
	}
	out, err := exec.Command(tool, "--file", file, "--group", group, "--key", "ProxyType", "1").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s ProxyType: %s: %w", tool, string(out), err)
	}
	return nil
}

func restoreKDE(tool string, snap linuxSnapshot) error {
	if tool == "" {
		return fmt.Errorf("kwriteconfig not found")
	}
	if err := restoreKDEConfigValue(tool, "ProxyType", snap.KDEProxyType); err != nil {
		return err
	}
	if err := restoreKDEConfigValue(tool, "httpProxy", snap.KDEHTTPProxy); err != nil {
		return err
	}
	return restoreKDEConfigValue(tool, "httpsProxy", snap.KDEHTTPSProxy)
}

func currentStateKDE(tool string) (SystemProxyCurrentState, error) {
	proxyType, err := kreadconfig(tool, "ProxyType")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpProxy, err := kreadconfig(tool, "httpProxy")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpsProxy, err := kreadconfig(tool, "httpsProxy")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	httpHost, httpPort := parseKDEProxyEndpoint(httpProxy)
	httpsHost, httpsPort := parseKDEProxyEndpoint(httpsProxy)
	enabled := strings.TrimSpace(proxyType) == "1" && httpHost != "" && httpHost == httpsHost && httpPort == httpsPort
	return SystemProxyCurrentState{Enabled: enabled, Addr: httpHost, Port: httpPort}, nil
}

func kdeConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kioslaverc"), nil
}

func readKDEConfigValue(path, group, key string) configValue {
	file, err := os.Open(path)
	if err != nil {
		return configValue{}
	}
	defer file.Close()

	inGroup := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inGroup = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]") == group
			continue
		}
		if !inGroup {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(name) == key {
			return configValue{Exists: true, Value: strings.TrimSpace(value)}
		}
	}
	return configValue{}
}

func restoreKDEConfigValue(tool, key string, value configValue) error {
	args := []string{"--file", "kioslaverc", "--group", "Proxy Settings", "--key", key}
	if value.Exists {
		args = append(args, value.Value)
	} else {
		args = append(args, "--delete")
	}
	out, err := exec.Command(tool, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s: %w", tool, key, string(out), err)
	}
	return nil
}

func kreadconfig(tool, key string) (string, error) {
	if tool == "" {
		return "", fmt.Errorf("kreadconfig not found")
	}
	out, err := exec.Command(tool, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", key).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %s: %w", tool, key, string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func parseKDEProxyEndpoint(value string) (string, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		parsed, err = url.Parse("http://" + value)
		if err != nil {
			return "", 0
		}
	}
	port, _ := strconv.Atoi(parsed.Port())
	return parsed.Hostname(), port
}
