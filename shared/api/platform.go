package api

type GetPlatformCapabilitiesRequest struct{}

type GetPlatformCapabilitiesReply struct {
	Capabilities []Capability `json:"capabilities"`
}

type GetPlatformCapabilitiesResult struct {
	Reply *GetPlatformCapabilitiesReply `json:"reply,omitempty"`
	Error *StructuredError              `json:"error,omitempty"`
}

type GetPrivilegedProviderStatusRequest struct{}

type GetPrivilegedProviderStatusReply struct {
	Status PrivilegedProviderStatus `json:"status"`
}

type GetPrivilegedProviderStatusResult struct {
	Reply *GetPrivilegedProviderStatusReply `json:"reply,omitempty"`
	Error *StructuredError                  `json:"error,omitempty"`
}

type PrivilegedProviderStatus struct {
	Installed       bool                `json:"installed"`
	Reachable       bool                `json:"reachable"`
	Authenticated   bool                `json:"authenticated"`
	Version         string              `json:"version,omitempty"`
	ExpectedVersion string              `json:"expected_version,omitempty"`
	Endpoint        string              `json:"endpoint,omitempty"`
	OwnerState      *ProviderOwnerState `json:"owner_state,omitempty"`
	Capabilities    []Capability        `json:"capabilities,omitempty"`
	Reason          string              `json:"reason,omitempty"`
}

type NetworkExtensionStatus struct {
	Installed    bool                `json:"installed"`
	Reachable    bool                `json:"reachable"`
	Authorized   bool                `json:"authorized"`
	BundleID     string              `json:"bundle_id,omitempty"`
	Version      string              `json:"version,omitempty"`
	OwnerState   *ProviderOwnerState `json:"owner_state,omitempty"`
	Capabilities []Capability        `json:"capabilities,omitempty"`
	Reason       string              `json:"reason,omitempty"`
}

type ProviderOwnerState struct {
	Owned           bool     `json:"owned"`
	Stale           bool     `json:"stale,omitempty"`
	UID             string   `json:"uid,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
	RuntimeID       string   `json:"runtime_id,omitempty"`
	SnapshotID      string   `json:"snapshot_id,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	StartedAt       int64    `json:"started_at,omitempty"`
	LastHeartbeatAt int64    `json:"last_heartbeat_at,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	RepairActions   []string `json:"repair_actions,omitempty"`
}

const (
	RuntimeModeMachineNetwork            = "machine_network"
	RuntimeModeAppleNetworkExtension     = "apple_network_extension"
	RepairActionClearMachineNetworkOwner = "clear_machine_network_owner"
)

type PrepareFeatureRequest struct {
	Feature string `json:"feature"`
}

type PrepareFeatureReply struct {
	Feature string          `json:"feature"`
	State   CapabilityState `json:"state"`
	Reason  string          `json:"reason,omitempty"`
}

type PrepareFeatureResult struct {
	Reply *PrepareFeatureReply `json:"reply,omitempty"`
	Error *StructuredError     `json:"error,omitempty"`
}

type RunRepairActionRequest struct {
	Action string `json:"action"`
}

type RunRepairActionReply struct {
	Action  string `json:"action"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
}

type RunRepairActionResult struct {
	Reply *RunRepairActionReply `json:"reply,omitempty"`
	Error *StructuredError      `json:"error,omitempty"`
}
