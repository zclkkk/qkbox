package api

type CapabilityState string

const (
	CapabilityAvailable   CapabilityState = "available"
	CapabilityUnavailable CapabilityState = "unavailable"
	CapabilityPartial     CapabilityState = "partial"
	CapabilityDegraded    CapabilityState = "degraded"
	CapabilityUnsupported CapabilityState = "unsupported"
)

type Capability struct {
	Name   string          `json:"name"`
	State  CapabilityState `json:"state"`
	Reason string          `json:"reason,omitempty"`
}

func RuntimeCapabilityShell() []Capability {
	return []Capability{
		{Name: "RUNTIME_LIFECYCLE", State: CapabilityAvailable},
		{Name: "STATUS_STREAM", State: CapabilityAvailable},
		{Name: "LOG_SOURCE", State: CapabilityAvailable},
		{Name: "TRAFFIC_SOURCE", State: CapabilityUnavailable, Reason: "Engine is not started."},
		{Name: "CONNECTION_SOURCE", State: CapabilityUnavailable, Reason: "Engine is not started."},
		{Name: "GROUP_SOURCE", State: CapabilityUnavailable, Reason: "Engine is not started."},
		{Name: "SELECT_OUTBOUND", State: CapabilityUnavailable, Reason: "Engine is not started."},
		{Name: "URL_TEST_SOURCE", State: CapabilityUnavailable, Reason: "Engine is not started."},
	}
}

func PlatformCapabilityShell() []Capability {
	return []Capability{
		{Name: "SYSTEM_PROXY", State: CapabilityUnavailable, Reason: "Implemented in Phase 7."},
		{Name: "TUN_MODE", State: CapabilityUnavailable, Reason: "Implemented in Phase 9."},
		{Name: "DNS_HIJACK", State: CapabilityUnavailable, Reason: "Implemented in Phase 9."},
		{Name: "BACKGROUND_SERVICE", State: CapabilityUnavailable, Reason: "Implemented in Phase 8."},
		{Name: "START_ON_BOOT", State: CapabilityUnavailable, Reason: "Implemented after privileged provider boundary."},
		{Name: "PROCESS_LOOKUP", State: CapabilityUnavailable, Reason: "Implemented when platform provider supports it."},
		{Name: "CONNECTION_TRACKING", State: CapabilityUnavailable, Reason: "Implemented when runtime observability supports it."},
	}
}
