//go:build windows

package capability

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"syscall"

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
	ProxyEnable   windowsDWORDValue  `json:"proxy_enable"`
	ProxyServer   windowsStringValue `json:"proxy_server"`
	AutoDetect    windowsDWORDValue  `json:"auto_detect"`
	AutoConfigURL windowsStringValue `json:"auto_config_url"`
}

type windowsDWORDValue struct {
	Exists bool   `json:"exists"`
	Value  uint32 `json:"value,omitempty"`
}

type windowsStringValue struct {
	Exists bool   `json:"exists"`
	Value  string `json:"value,omitempty"`
}

func (p *windowsSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("open internet settings: %w", err)
	}
	defer k.Close()

	proxyEnable, err := readDWORDValue(k, "ProxyEnable")
	if err != nil {
		return nil, fmt.Errorf("read ProxyEnable: %w", err)
	}
	proxyServer, err := readStringValue(k, "ProxyServer")
	if err != nil {
		return nil, fmt.Errorf("read ProxyServer: %w", err)
	}
	autoDetect, err := readDWORDValue(k, "AutoDetect")
	if err != nil {
		return nil, fmt.Errorf("read AutoDetect: %w", err)
	}
	autoConfigURL, err := readStringValue(k, "AutoConfigURL")
	if err != nil {
		return nil, fmt.Errorf("read AutoConfigURL: %w", err)
	}

	snap := windowsSnapshot{
		ProxyEnable:   proxyEnable,
		ProxyServer:   proxyServer,
		AutoDetect:    autoDetect,
		AutoConfigURL: autoConfigURL,
	}

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

	if err := restoreDWORDValue(k, "ProxyEnable", snap.ProxyEnable); err != nil {
		return fmt.Errorf("restore ProxyEnable: %w", err)
	}
	if err := restoreStringValue(k, "ProxyServer", snap.ProxyServer); err != nil {
		return fmt.Errorf("restore ProxyServer: %w", err)
	}
	if err := restoreDWORDValue(k, "AutoDetect", snap.AutoDetect); err != nil {
		return fmt.Errorf("restore AutoDetect: %w", err)
	}
	if err := restoreStringValue(k, "AutoConfigURL", snap.AutoConfigURL); err != nil {
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
	autoDetect, _, _ := k.GetIntegerValue("AutoDetect")
	autoConfigURL, _, _ := k.GetStringValue("AutoConfigURL")

	state := SystemProxyCurrentState{Enabled: enabled64 != 0 && autoDetect == 0 && autoConfigURL == ""}
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

func readDWORDValue(k registry.Key, name string) (windowsDWORDValue, error) {
	value, _, err := k.GetIntegerValue(name)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return windowsDWORDValue{}, nil
		}
		return windowsDWORDValue{}, err
	}
	return windowsDWORDValue{Exists: true, Value: uint32(value)}, nil
}

func readStringValue(k registry.Key, name string) (windowsStringValue, error) {
	value, _, err := k.GetStringValue(name)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return windowsStringValue{}, nil
		}
		return windowsStringValue{}, err
	}
	return windowsStringValue{Exists: true, Value: value}, nil
}

func restoreDWORDValue(k registry.Key, name string, value windowsDWORDValue) error {
	if value.Exists {
		return k.SetDWordValue(name, value.Value)
	}
	return deleteRegistryValueIfPresent(k, name)
}

func restoreStringValue(k registry.Key, name string, value windowsStringValue) error {
	if value.Exists {
		return k.SetStringValue(name, value.Value)
	}
	return deleteRegistryValueIfPresent(k, name)
}

func deleteRegistryValueIfPresent(k registry.Key, name string) error {
	if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
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
