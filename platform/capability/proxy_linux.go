//go:build linux

package capability

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type linuxSystemProxy struct {
	availability SystemProxyAvailability
}

func NewSystemProxyProvider() SystemProxyProvider {
	p := &linuxSystemProxy{}
	p.availability = p.detectAvailability()
	return p
}

func (p *linuxSystemProxy) Availability() SystemProxyAvailability {
	return p.availability
}

func (p *linuxSystemProxy) detectAvailability() SystemProxyAvailability {
	if isGNOME() {
		return SystemProxyAvailability{Available: true, Supported: true}
	}
	if kdeTool() != "" {
		return SystemProxyAvailability{Available: true, Supported: true}
	}
	return SystemProxyAvailability{
		Available: true,
		Supported: false,
		Reason:    "No supported desktop environment detected (GNOME with gsettings or KDE with kwriteconfig required).",
	}
}

type linuxSnapshot struct {
	DE             string `json:"de"`
	GNOMEMode      string `json:"gnome_mode,omitempty"`
	GNOMEHTTPHost  string `json:"gnome_http_host,omitempty"`
	GNOMEHTTPPort  int    `json:"gnome_http_port,omitempty"`
	GNOMEHTTPSHost string `json:"gnome_https_host,omitempty"`
	GNOMEHTTPSPort int    `json:"gnome_https_port,omitempty"`
}

func (p *linuxSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	if isGNOME() {
		return snapshotGNOME()
	}
	if tool := kdeTool(); tool != "" {
		return snapshotKDE(tool)
	}
	return nil, fmt.Errorf("no supported desktop environment")
}

func (p *linuxSystemProxy) Apply(addr string, port int) error {
	if isGNOME() {
		return applyGNOME(addr, port)
	}
	if tool := kdeTool(); tool != "" {
		return applyKDE(tool, addr, port)
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
		return restoreKDE(snap)
	}
	return fmt.Errorf("unknown DE in snapshot: %s", snap.DE)
}

func (p *linuxSystemProxy) CurrentState() (SystemProxyCurrentState, error) {
	if isGNOME() {
		return currentStateGNOME()
	}
	return SystemProxyCurrentState{}, nil
}

// GNOME

func isGNOME() bool {
	_, err := exec.LookPath("gsettings")
	return err == nil
}

func gsettingsGet(schema, key string) string {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gsettingsSet(schema, key, value string) error {
	out, err := exec.Command("gsettings", "set", schema, key, value).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gsettings set %s %s %s: %s: %w", schema, key, value, string(out), err)
	}
	return nil
}

func snapshotGNOME() (*SystemProxySnapshot, error) {
	mode := gsettingsGet("org.gnome.system.proxy", "mode")
	httpHost := gsettingsGet("org.gnome.system.proxy.http", "host")
	httpPort := gsettingsGet("org.gnome.system.proxy.http", "port")
	httpsHost := gsettingsGet("org.gnome.system.proxy.https", "host")
	httpsPort := gsettingsGet("org.gnome.system.proxy.https", "port")

	httpPortInt, _ := strconv.Atoi(httpPort)
	httpsPortInt, _ := strconv.Atoi(httpsPort)

	snap := linuxSnapshot{
		DE:             "gnome",
		GNOMEMode:      mode,
		GNOMEHTTPHost:  httpHost,
		GNOMEHTTPPort:  httpPortInt,
		GNOMEHTTPSHost: httpsHost,
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
	if err := gsettingsSet("org.gnome.system.proxy.http", "host", "'"+addr+"'"); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "port", portStr); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "host", "'"+addr+"'"); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "port", portStr); err != nil {
		return err
	}
	return gsettingsSet("org.gnome.system.proxy", "mode", "'manual'")
}

func restoreGNOME(snap linuxSnapshot) error {
	if snap.GNOMEMode == "" || snap.GNOMEMode == "'none'" {
		return gsettingsSet("org.gnome.system.proxy", "mode", "'none'")
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "host", "'"+snap.GNOMEHTTPHost+"'"); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.http", "port", strconv.Itoa(snap.GNOMEHTTPPort)); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "host", "'"+snap.GNOMEHTTPSHost+"'"); err != nil {
		return err
	}
	if err := gsettingsSet("org.gnome.system.proxy.https", "port", strconv.Itoa(snap.GNOMEHTTPSPort)); err != nil {
		return err
	}
	return gsettingsSet("org.gnome.system.proxy", "mode", snap.GNOMEMode)
}

func currentStateGNOME() (SystemProxyCurrentState, error) {
	mode := gsettingsGet("org.gnome.system.proxy", "mode")
	enabled := mode == "'manual'"
	host := strings.Trim(gsettingsGet("org.gnome.system.proxy.http", "host"), "'")
	portStr := gsettingsGet("org.gnome.system.proxy.http", "port")
	port, _ := strconv.Atoi(portStr)
	return SystemProxyCurrentState{Enabled: enabled, Addr: host, Port: port}, nil
}

// KDE

func kdeTool() string {
	if _, err := exec.LookPath("kwriteconfig6"); err == nil {
		return "kwriteconfig6"
	}
	if _, err := exec.LookPath("kwriteconfig5"); err == nil {
		return "kwriteconfig5"
	}
	return ""
}

func snapshotKDE(tool string) (*SystemProxySnapshot, error) {
	snap := linuxSnapshot{DE: "kde"}
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

func restoreKDE(snap linuxSnapshot) error {
	tool := kdeTool()
	if tool == "" {
		return fmt.Errorf("kwriteconfig not found")
	}
	group := "Proxy Settings"
	file := "kioslaverc"
	out, err := exec.Command(tool, "--file", file, "--group", group, "--key", "ProxyType", "0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s ProxyType: %s: %w", tool, string(out), err)
	}
	return nil
}
