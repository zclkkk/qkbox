//go:build darwin

package capability

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type darwinSystemProxy struct{}

func NewSystemProxyProvider() SystemProxyProvider {
	return &darwinSystemProxy{}
}

func (p *darwinSystemProxy) Availability() SystemProxyAvailability {
	return SystemProxyAvailability{Available: true, Supported: true}
}

type darwinSnapshot struct {
	Interface      string `json:"interface"`
	WebProxy       bool   `json:"web_proxy"`
	WebProxyHost   string `json:"web_proxy_host"`
	WebProxyPort   int    `json:"web_proxy_port"`
	SecureProxy    bool   `json:"secure_proxy"`
	SecureProxyHost string `json:"secure_proxy_host"`
	SecureProxyPort int    `json:"secure_proxy_port"`
}

func (p *darwinSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	iface, err := activeInterface()
	if err != nil {
		return nil, fmt.Errorf("detect active interface: %w", err)
	}
	snap := darwinSnapshot{Interface: iface}

	webEnabled, webHost, webPort, err := readProxyState(iface, "webproxy")
	if err != nil {
		return nil, err
	}
	snap.WebProxy = webEnabled
	snap.WebProxyHost = webHost
	snap.WebProxyPort = webPort

	secEnabled, secHost, secPort, err := readProxyState(iface, "securewebproxy")
	if err != nil {
		return nil, err
	}
	snap.SecureProxy = secEnabled
	snap.SecureProxyHost = secHost
	snap.SecureProxyPort = secPort

	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return &SystemProxySnapshot{Raw: raw}, nil
}

func (p *darwinSystemProxy) Apply(addr string, port int) error {
	iface, err := activeInterface()
	if err != nil {
		return fmt.Errorf("detect active interface: %w", err)
	}
	portStr := fmt.Sprintf("%d", port)

	if err := runNetworksetup("-setwebproxy", iface, addr, portStr); err != nil {
		return fmt.Errorf("set web proxy: %w", err)
	}
	if err := runNetworksetup("-setsecurewebproxy", iface, addr, portStr); err != nil {
		return fmt.Errorf("set secure web proxy: %w", err)
	}
	return nil
}

func (p *darwinSystemProxy) Restore(snapshot *SystemProxySnapshot) error {
	var snap darwinSnapshot
	if err := json.Unmarshal(snapshot.Raw, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}

	if snap.WebProxy {
		if err := runNetworksetup("-setwebproxy", snap.Interface, snap.WebProxyHost, fmt.Sprintf("%d", snap.WebProxyPort)); err != nil {
			return err
		}
	} else {
		if err := runNetworksetup("-setwebproxystate", snap.Interface, "off"); err != nil {
			return err
		}
	}

	if snap.SecureProxy {
		if err := runNetworksetup("-setsecurewebproxy", snap.Interface, snap.SecureProxyHost, fmt.Sprintf("%d", snap.SecureProxyPort)); err != nil {
			return err
		}
	} else {
		if err := runNetworksetup("-setsecurewebproxystate", snap.Interface, "off"); err != nil {
			return err
		}
	}
	return nil
}

func (p *darwinSystemProxy) CurrentState() (SystemProxyCurrentState, error) {
	iface, err := activeInterface()
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	enabled, host, port, err := readProxyState(iface, "webproxy")
	if err != nil {
		return SystemProxyCurrentState{}, err
	}
	return SystemProxyCurrentState{Enabled: enabled, Addr: host, Port: port}, nil
}

func activeInterface() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", err
	}
	var iface string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			iface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
			break
		}
	}
	if iface == "" {
		return "", fmt.Errorf("no default interface found")
	}

	hwOut, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return "", err
	}
	var displayName string
	var currentDevice string
	for _, line := range strings.Split(string(hwOut), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Hardware Port:") {
			displayName = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
		}
		if strings.HasPrefix(line, "Device:") {
			currentDevice = strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if currentDevice == iface {
				return displayName, nil
			}
		}
	}
	return iface, nil
}

func readProxyState(iface, proxyType string) (enabled bool, host string, port int, err error) {
	stateOut, err := exec.Command("networksetup", "-get"+proxyType, iface).Output()
	if err != nil {
		return false, "", 0, err
	}
	lines := strings.Split(string(stateOut), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Enabled:") {
			enabled = strings.TrimSpace(strings.TrimPrefix(line, "Enabled:")) == "Yes"
		}
		if strings.HasPrefix(line, "Server:") {
			host = strings.TrimSpace(strings.TrimPrefix(line, "Server:"))
		}
		if strings.HasPrefix(line, "Port:") {
			portStr := strings.TrimSpace(strings.TrimPrefix(line, "Port:"))
			fmt.Sscanf(portStr, "%d", &port)
		}
	}
	return
}

func runNetworksetup(args ...string) error {
	out, err := exec.Command("networksetup", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("networksetup %s: %s: %w", strings.Join(args, " "), string(out), err)
	}
	return nil
}
