package model

type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityInfo    DiagnosticSeverity = "info"
)

type ValidationStatus string

const (
	ValidationStatusValid   ValidationStatus = "valid"
	ValidationStatusInvalid ValidationStatus = "invalid"
	ValidationStatusUnknown ValidationStatus = "unknown"
)

type ValidationDiagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Field    string             `json:"field,omitempty"`
	Message  string             `json:"message"`
}

type Diagnostics struct {
	ProfileID       string                 `json:"profile_id"`
	Status          ValidationStatus       `json:"status"`
	Entries         []ValidationDiagnostic `json:"entries"`
	RedactedPreview string                 `json:"redacted_preview,omitempty"`
}
