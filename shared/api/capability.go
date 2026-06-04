package api

type CapabilityState string

const (
	CapabilityAvailable   CapabilityState = "available"
	CapabilityUnavailable CapabilityState = "unavailable"
	CapabilityPartial     CapabilityState = "partial"
	CapabilityDegraded    CapabilityState = "degraded"
	CapabilityUnsupported CapabilityState = "unsupported"
)

const (
	CapabilitySystemProxy        = "SYSTEM_PROXY"
	CapabilityTunMode            = "TUN_MODE"
	CapabilityDNSHijack          = "DNS_HIJACK"
	CapabilityBackgroundService  = "BACKGROUND_SERVICE"
	CapabilityStartOnBoot        = "START_ON_BOOT"
	CapabilityProcessLookup      = "PROCESS_LOOKUP"
	CapabilityConnectionTracking = "CONNECTION_TRACKING"
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
		{Name: CapabilitySystemProxy, State: CapabilityUnavailable, Reason: "System proxy provider is unavailable."},
		{Name: CapabilityTunMode, State: CapabilityUnavailable, Reason: "Exclusive network mode is unavailable."},
		{Name: CapabilityDNSHijack, State: CapabilityUnavailable, Reason: "DNS hijack mode is unavailable."},
		{Name: CapabilityBackgroundService, State: CapabilityUnavailable, Reason: "Privileged provider is unavailable."},
		{Name: CapabilityStartOnBoot, State: CapabilityUnavailable, Reason: "Start-on-boot provider is unavailable."},
		{Name: CapabilityProcessLookup, State: CapabilityUnavailable, Reason: "Process lookup source is unavailable."},
		{Name: CapabilityConnectionTracking, State: CapabilityUnavailable, Reason: "Connection tracking source is unavailable."},
	}
}
