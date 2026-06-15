package uriparse

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseSupportedProtocols(t *testing.T) {
	vmessPayload, err := json.Marshal(map[string]any{
		"ps":   "vmess node",
		"add":  "vmess.example.com",
		"port": "443",
		"id":   "00000000-0000-0000-0000-000000000001",
		"aid":  "0",
		"scy":  "auto",
		"net":  "ws",
		"path": "/ws",
		"host": "cdn.example.com",
		"tls":  "tls",
		"sni":  "vmess.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret@ss.example.com:8388")) + "#ss%20node",
		"vmess://" + base64.StdEncoding.EncodeToString(vmessPayload),
		"vless://00000000-0000-0000-0000-000000000002@vless.example.com:443?encryption=none&security=tls&type=ws&host=cdn.example.com&path=%2Fedge#vless%20node",
		"trojan://secret@trojan.example.com:443?sni=trojan.example.com#trojan%20node",
		"hysteria2://secret@hy2.example.com:443?sni=hy2.example.com#hy2%20node",
	}

	outbounds, err := Parse(strings.Join(lines, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(outbounds) != len(lines) {
		t.Fatalf("count = %d", len(outbounds))
	}
	assertOutbound(t, outbounds[0], "ss node", "shadowsocks")
	assertOutbound(t, outbounds[1], "vmess node", "vmess")
	assertOutbound(t, outbounds[2], "vless node", "vless")
	assertOutbound(t, outbounds[3], "trojan node", "trojan")
	assertOutbound(t, outbounds[4], "hy2 node", "hysteria2")
	if outbounds[0].Outbound["method"] != "aes-128-gcm" || outbounds[0].Outbound["password"] != "secret" {
		t.Fatalf("shadowsocks outbound = %+v", outbounds[0].Outbound)
	}
	if outbounds[2].Outbound["transport"] == nil {
		t.Fatalf("vless transport missing: %+v", outbounds[2].Outbound)
	}
	if outbounds[3].Outbound["tls"] == nil || outbounds[4].Outbound["tls"] == nil {
		t.Fatalf("tls missing: trojan=%+v hy2=%+v", outbounds[3].Outbound, outbounds[4].Outbound)
	}
}

func TestParseDeduplicatesTags(t *testing.T) {
	line := "ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret@ss.example.com:8388")) + "#same"
	outbounds, err := Parse(line + "\n" + line)
	if err != nil {
		t.Fatal(err)
	}
	if outbounds[0].Tag != "same" || outbounds[1].Tag != "same-2" {
		t.Fatalf("tags = %s, %s", outbounds[0].Tag, outbounds[1].Tag)
	}
}

func TestParseRejectsUnsupportedAndMalformedURIs(t *testing.T) {
	for _, input := range []string{
		"http://example.com",
		"ss://not-base64",
		"vless://missing-port@example.com",
	} {
		if _, err := Parse(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func assertOutbound(t *testing.T, outbound ParsedOutbound, tag string, typ string) {
	t.Helper()
	if outbound.Tag != tag || outbound.Type != typ {
		t.Fatalf("outbound = %+v, want tag=%s type=%s", outbound, tag, typ)
	}
	if outbound.Outbound["tag"] != tag || outbound.Outbound["type"] != typ {
		t.Fatalf("outbound map = %+v", outbound.Outbound)
	}
}
