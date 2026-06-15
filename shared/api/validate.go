package api

import "github.com/zclkkk/qkbox/shared/model"

type ValidateProfileContentRequest struct {
	ProfileID string `json:"profile_id,omitempty"`
	Content   string `json:"content"`
}

type ValidateProfileContentReply struct {
	Diagnostics model.Diagnostics `json:"diagnostics"`
}

type ValidateProfileContentResult struct {
	Reply *ValidateProfileContentReply `json:"reply,omitempty"`
	Error *StructuredError             `json:"error,omitempty"`
}
