package model

type Snapshot struct {
	ID                    string               `json:"id"`
	ProfileID             string               `json:"profile_id"`
	ValidationStatus      ValidationStatus     `json:"validation_status"`
	Diagnostics           []ValidationDiagnostic `json:"diagnostics,omitempty"`
	RuntimeSummary        *RuntimeSummary      `json:"runtime_summary,omitempty"`
	RequiredCapabilities  []string             `json:"required_capabilities,omitempty"`
	CreatedAt             int64                `json:"created_at"`
}

type SnapshotSummary struct {
	ID               string           `json:"id"`
	ProfileID        string           `json:"profile_id"`
	ValidationStatus ValidationStatus `json:"validation_status"`
	CreatedAt        int64            `json:"created_at"`
}

type ValidationStatus string

const (
	ValidationStatusValid   ValidationStatus = "valid"
	ValidationStatusInvalid ValidationStatus = "invalid"
	ValidationStatusUnknown ValidationStatus = "unknown"
)

type RuntimeSummary struct {
	InboundTypes  []string `json:"inbound_types,omitempty"`
	OutboundTypes []string `json:"outbound_types,omitempty"`
	Protocols     []string `json:"protocols,omitempty"`
}
