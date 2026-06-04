//go:build windows

package capability

import (
	"encoding/json"
	"fmt"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type windowsSystemProxy struct{}

func NewSystemProxyProvider() SystemProxyProvider {
	return &windowsSystemProxy{}
}

func (p *windowsSystemProxy) Availability() SystemProxyAvailability {
	return SystemProxyAvailability{Available: true, Supported: true}
}

const internetSettingsPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type windowsSnapshot struct {
	ProxyEnable   uint32 `json:"proxy_enable"`
	ProxyServer   string `json:"proxy_server"`
	AutoDetect    uint32 `json:"auto_detect"`
	AutoConfigURL string `json:"auto_config_url"`
}

func (p *windowsSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("open internet settings: %w", err)
	}
	defer k.Close()

	snap := windowsSnapshot{}

	snap.ProxyEnable, _, _ = k.GetIntegerValue("ProxyEnable")
	snap.ProxyServer, _, _ = k.GetStringValue("ProxyServer")
	snap.AutoDetect, _, _ = k.GetIntegerValue("AutoDetect")
	snap.AutoConfigURL, _, _ = k.GetStringValue("AutoConfigURL")

	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return &SystemProxySnapshot{Raw: raw}, nil
}

func (p *windowsSystemProxy) Apply(addr string, port int) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("open internet settings: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	if err := k.SetStringValue("ProxyServer", addr+":"+strconv.Itoa(port)); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}
	if err := k.SetDWordValue("AutoDetect", 0); err != nil {
		return fmt.Errorf("set AutoDetect: %w", err)
	}
	if err := k.SetStringValue("AutoConfigURL", ""); err != nil {
		return fmt.Errorf("set AutoConfigURL: %w", err)
	}

	notifySystemProxyChanged()
	return nil
}

func (p *windowsSystemProxy) Restore(snapshot *SystemProxySnapshot) error {
	var snap windowsSnapshot
	if err := json.Unmarshal(snapshot.Raw, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("open internet settings: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", snap.ProxyEnable); err != nil {
		return fmt.Errorf("restore ProxyEnable: %w", err)
	}
	if err := k.SetStringValue("ProxyServer", snap.ProxyServer); err != nil {
		return fmt.Errorf("restore ProxyServer: %w", err)
	}
	if err := k.SetDWordValue("AutoDetect", snap.AutoDetect); err != nil {
		return fmt.Errorf("restore AutoDetect: %w", err)
	}
	if err := k.SetStringValue("AutoConfigURL", snap.AutoConfigURL); err != nil {
		return fmt.Errorf("restore AutoConfigURL: %w", err)
	}

	notifySystemProxyChanged()
	return nil
}

func (p *windowsSystemProxy) CurrentState() (SystemProxyCurrentState, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.READ)
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	defer k.Close()

	enabled64, _, _ := k.GetIntegerValue("ProxyEnable")
	server, _, _ := k.GetStringValue("ProxyServer")

	state := SystemProxyCurrentState{Enabled: enabled64 != 0}
	if server != "" {
		host, portStr, err := splitHostPort(server)
		if err == nil {
			state.Addr = host
			state.Port, _ = strconv.Atoi(portStr)
		}
	}
	return state, nil
}

func splitHostPort(s string) (string, string, error) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return s[:i], s[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("missing port in %q", s)
}

func notifySystemProxyChanged() {
	mod := syscall.NewLazyDLL("wininet.dll")
	proc := mod.NewProc("InternetSetOptionW")
	if proc.Find() != nil {
		return
	}
	INTERNET_OPTION_SETTINGS_CHANGED := 39
	proc.Call(0, uintptr(INTERNET_OPTION_SETTINGS_CHANGED), 0, 0)
}
