package api

import "github.com/zclkkk/qkbox/shared/model"

type CreateProfileRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CreateProfileReply struct {
	Profile model.Profile `json:"profile"`
}

type CreateProfileResult struct {
	Reply *CreateProfileReply `json:"reply,omitempty"`
	Error *StructuredError    `json:"error,omitempty"`
}

type UpdateProfileDraftRequest struct {
	ProfileID string `json:"profile_id"`
	Content   string `json:"content"`
}

type UpdateProfileDraftReply struct {
	Profile model.Profile `json:"profile"`
}

type UpdateProfileDraftResult struct {
	Reply *UpdateProfileDraftReply `json:"reply,omitempty"`
	Error *StructuredError         `json:"error,omitempty"`
}

type DeleteProfileRequest struct {
	ProfileID string `json:"profile_id"`
}

type DeleteProfileReply struct{}

type DeleteProfileResult struct {
	Reply *DeleteProfileReply `json:"reply,omitempty"`
	Error *StructuredError    `json:"error,omitempty"`
}

type ListProfilesRequest struct{}

type ListProfilesReply struct {
	Profiles []model.ProfileSummary `json:"profiles"`
}

type ListProfilesResult struct {
	Reply *ListProfilesReply `json:"reply,omitempty"`
	Error *StructuredError   `json:"error,omitempty"`
}

type GetProfileRequest struct {
	ProfileID string `json:"profile_id"`
}

type GetProfileReply struct {
	Profile model.Profile `json:"profile"`
	Content string        `json:"content,omitempty"`
}

type GetProfileResult struct {
	Reply *GetProfileReply `json:"reply,omitempty"`
	Error *StructuredError `json:"error,omitempty"`
}
