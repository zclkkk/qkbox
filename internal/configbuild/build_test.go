package configbuild

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/internal/uriparse"
	"github.com/zclkkk/qkbox/shared/model"
)

func TestBuildWithoutNodesCreatesDirectConfig(t *testing.T) {
	config, err := Build(nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertValidSingboxConfig(t, config)

	var decoded map[string]any
	if err := json.Unmarshal([]byte(config), &decoded); err != nil {
		t.Fatal(err)
	}
	route := decoded["route"].(map[string]any)
	if route["final"] != "direct" {
		t.Fatalf("route = %+v", route)
	}
}

func TestBuildWithNodesCreatesSelectorConfig(t *testing.T) {
	nodes := parseTestNodes(t)
	config, err := Build(nodes, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertValidSingboxConfig(t, config)

	var decoded struct {
		Outbounds []map[string]any `json:"outbounds"`
		Route     map[string]any   `json:"route"`
	}
	if err := json.Unmarshal([]byte(config), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Outbounds[0]["type"] != "selector" || decoded.Outbounds[0]["tag"] != "proxy" {
		t.Fatalf("first outbound = %+v", decoded.Outbounds[0])
	}
	if decoded.Route["final"] != "proxy" {
		t.Fatalf("route = %+v", decoded.Route)
	}
}

func TestBuildModesCreateValidConfigs(t *testing.T) {
	nodes := parseTestNodes(t)
	for _, mode := range []Mode{ModeRule, ModeGlobal, ModeDirect} {
		config, err := Build(nodes, Options{Mode: mode})
		if err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
		assertValidSingboxConfig(t, config)
	}
}

func TestBuildParsedSupportedProtocolsCreatesValidConfig(t *testing.T) {
	vmessPayload, err := json.Marshal(map[string]any{
		"ps":   "vmess",
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
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret@ss.example.com:8388")) + "#ss",
		"vmess://" + base64.StdEncoding.EncodeToString(vmessPayload),
		"vless://00000000-0000-0000-0000-000000000002@vless.example.com:443?encryption=none&security=tls&type=ws&host=cdn.example.com&path=%2Fedge#vless",
		"trojan://secret@trojan.example.com:443?sni=trojan.example.com#trojan",
	}
	nodes, err := uriparse.Parse(strings.Join(lines, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	config, err := Build(nodes, Options{})
	if err != nil {
		t.Fatal(err)
	}
	assertValidSingboxConfig(t, config)
}

func parseTestNodes(t *testing.T) []uriparse.ParsedOutbound {
	t.Helper()
	line := "ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:secret@ss.example.com:8388")) + "#ss"
	nodes, err := uriparse.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	return nodes
}

func assertValidSingboxConfig(t *testing.T, config string) {
	t.Helper()
	diag := singboxadapter.Validate(config)
	if diag.Status != model.ValidationStatusValid {
		t.Fatalf("invalid config: %+v\n%s", diag, config)
	}
}
