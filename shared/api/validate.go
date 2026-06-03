package api

import "github.com/zclkkk/qkbox/shared/model"

type ValidateProfileDraftRequest struct {
	ProfileID string `json:"profile_id"`
}

type ValidateProfileDraftReply struct {
	Diagnostics model.Diagnostics `json:"diagnostics"`
}

type ValidateProfileDraftResult struct {
	Reply *ValidateProfileDraftReply `json:"reply,omitempty"`
	Error *StructuredError           `json:"error,omitempty"`
}

type GetProfileDiagnosticsRequest struct {
	ProfileID string `json:"profile_id"`
}

type GetProfileDiagnosticsReply struct {
	Diagnostics model.Diagnostics `json:"diagnostics"`
}

type GetProfileDiagnosticsResult struct {
	Reply *GetProfileDiagnosticsReply `json:"reply,omitempty"`
	Error *StructuredError            `json:"error,omitempty"`
}
