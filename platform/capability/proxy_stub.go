//go:build !darwin && !windows && !linux

package capability

import (
	"fmt"
	"runtime"
)

type stubSystemProxy struct{}

func NewSystemProxyProvider() SystemProxyProvider {
	return &stubSystemProxy{}
}

func (p *stubSystemProxy) Availability() SystemProxyAvailability {
	return SystemProxyAvailability{
		Available: false,
		Reason:    fmt.Sprintf("System proxy is not supported on %s.", runtime.GOOS),
	}
}

func (p *stubSystemProxy) Snapshot() (*SystemProxySnapshot, error) {
	return nil, fmt.Errorf("system proxy not supported")
}

func (p *stubSystemProxy) Apply(addr string, port int) error {
	return fmt.Errorf("system proxy not supported")
}

func (p *stubSystemProxy) Restore(snapshot *SystemProxySnapshot) error {
	return fmt.Errorf("system proxy not supported")
}

func (p *stubSystemProxy) CurrentState() (SystemProxyCurrentState, error) {
	return SystemProxyCurrentState{}, nil
}
