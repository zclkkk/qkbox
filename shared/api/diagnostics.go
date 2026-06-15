package api

import "github.com/zclkkk/qkbox/shared/model"

type GetDiagnosticsReportRequest struct{}

type GetDiagnosticsReportReply struct {
	Report ProductDiagnosticsReport `json:"report"`
}

type GetDiagnosticsReportResult struct {
	Reply *GetDiagnosticsReportReply `json:"reply,omitempty"`
	Error *StructuredError           `json:"error,omitempty"`
}

type CreateDebugBundleRequest struct{}

type CreateDebugBundleReply struct {
	BundlePath string                   `json:"bundle_path"`
	Manifest   DebugBundleManifest      `json:"manifest"`
	Report     ProductDiagnosticsReport `json:"report"`
}

type CreateDebugBundleResult struct {
	Reply *CreateDebugBundleReply `json:"reply,omitempty"`
	Error *StructuredError        `json:"error,omitempty"`
}

type ProductDiagnosticsReport struct {
	GeneratedAt              int64                           `json:"generated_at"`
	APIVersion               string                          `json:"api_version"`
	MinSupportedAPIVersion   string                          `json:"min_supported_api_version"`
	SchemaRevision           string                          `json:"schema_revision"`
	DBSchemaVersion          int                             `json:"db_schema_version"`
	AppVersion               string                          `json:"app_version"`
	QKBoxDVersion            string                          `json:"qkboxd_version"`
	Platform                 model.Platform                  `json:"platform"`
	EngineStatus             EngineStatus                    `json:"engine_status"`
	RuntimeCapabilities      []Capability                    `json:"runtime_capabilities"`
	PlatformCapabilities     []Capability                    `json:"platform_capabilities"`
	PrivilegedProviderStatus PrivilegedProviderStatus        `json:"privileged_provider_status"`
	NetworkExtensionStatus   *NetworkExtensionStatus         `json:"network_extension_status,omitempty"`
	SystemProxyStatus        *GetSystemProxyStatusReply      `json:"system_proxy_status,omitempty"`
	SystemProxyError         *StructuredError                `json:"system_proxy_error,omitempty"`
	ActiveProfileID          string                          `json:"active_profile_id,omitempty"`
	Subscriptions            []DiagnosticSubscriptionSummary `json:"subscriptions"`
	DataAssets               []DiagnosticAssetSummary        `json:"data_assets"`
	Checks                   []DiagnosticCheck               `json:"checks"`
}

type DiagnosticSubscriptionSummary struct {
	ID               string `json:"id"`
	ProfileID        string `json:"profile_id"`
	Name             string `json:"name"`
	RedactedURL      string `json:"redacted_url"`
	UpdatePolicy     string `json:"update_policy"`
	LastStatus       string `json:"last_status"`
	LastErrorCode    string `json:"last_error_code,omitempty"`
	LastErrorMessage string `json:"last_error_message,omitempty"`
	LastCheckedAt    int64  `json:"last_checked_at,omitempty"`
	LastUpdatedAt    int64  `json:"last_updated_at,omitempty"`
	ContentSHA256    string `json:"content_sha256,omitempty"`
}

type DiagnosticAssetSummary struct {
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	RedactedSourceURL string `json:"redacted_source_url"`
	Status            string `json:"status"`
	CacheKey          string `json:"cache_key,omitempty"`
	Version           string `json:"version,omitempty"`
	ContentSHA256     string `json:"content_sha256,omitempty"`
	SizeBytes         int64  `json:"size_bytes,omitempty"`
	LastErrorCode     string `json:"last_error_code,omitempty"`
	LastErrorMessage  string `json:"last_error_message,omitempty"`
	LastCheckedAt     int64  `json:"last_checked_at,omitempty"`
	LastUpdatedAt     int64  `json:"last_updated_at,omitempty"`
}

type DiagnosticCheck struct {
	Name     string          `json:"name"`
	State    CapabilityState `json:"state"`
	Reason   string          `json:"reason,omitempty"`
	Recovery string          `json:"recovery,omitempty"`
}

type DebugBundleManifest struct {
	CreatedAt      int64    `json:"created_at"`
	APIVersion     string   `json:"api_version"`
	SchemaRevision string   `json:"schema_revision"`
	AppVersion     string   `json:"app_version"`
	QKBoxDVersion  string   `json:"qkboxd_version"`
	Platform       string   `json:"platform"`
	Files          []string `json:"files"`
	Redaction      string   `json:"redaction"`
}
