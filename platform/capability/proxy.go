package capability

import "encoding/json"

// SystemProxyProvider manages OS-level HTTP proxy settings.
// Implementations are platform-specific (build-tagged).
type SystemProxyProvider interface {
	// Availability returns whether the platform supports system proxy.
	// On Linux, this also reports whether a supported DE/tool was detected.
	Availability() SystemProxyAvailability

	// Snapshot captures the current OS proxy state for later Restore.
	Snapshot() (*SystemProxySnapshot, error)

	// Apply sets the system HTTP proxy to addr:port.
	// It modifies only the fields documented for the platform.
	Apply(addr string, port int) error

	// Restore reverts the OS proxy state to what was captured by Snapshot.
	Restore(snapshot *SystemProxySnapshot) error

	// CurrentState reads what the OS proxy is currently set to.
	CurrentState() (SystemProxyCurrentState, error)
}

type SystemProxyAvailability struct {
	Available bool
	Supported bool
	Reason    string
}

type SystemProxyCurrentState struct {
	Enabled bool
	Addr    string
	Port    int
}

type SystemProxySnapshot struct {
	Raw json.RawMessage
}
