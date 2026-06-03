package api

import "github.com/zclkkk/qkbox/shared/model"

type CreateProfileSnapshotRequest struct {
	ProfileID string `json:"profile_id"`
}

type CreateProfileSnapshotReply struct {
	Snapshot model.Snapshot `json:"snapshot"`
}

type CreateProfileSnapshotResult struct {
	Reply *CreateProfileSnapshotReply `json:"reply,omitempty"`
	Error *StructuredError            `json:"error,omitempty"`
}

type ActivateProfileSnapshotRequest struct {
	SnapshotID string `json:"snapshot_id"`
}

type ActivateProfileSnapshotReply struct{}

type ActivateProfileSnapshotResult struct {
	Reply *ActivateProfileSnapshotReply `json:"reply,omitempty"`
	Error *StructuredError              `json:"error,omitempty"`
}

type GetActiveProfileRequest struct{}

type GetActiveProfileReply struct {
	Profile *model.Profile `json:"profile,omitempty"`
}

type GetActiveProfileResult struct {
	Reply *GetActiveProfileReply `json:"reply,omitempty"`
	Error *StructuredError       `json:"error,omitempty"`
}

type GetActiveSnapshotRequest struct{}

type GetActiveSnapshotReply struct {
	Snapshot *model.Snapshot `json:"snapshot,omitempty"`
}

type GetActiveSnapshotResult struct {
	Reply *GetActiveSnapshotReply `json:"reply,omitempty"`
	Error *StructuredError        `json:"error,omitempty"`
}

type ListSnapshotsRequest struct {
	ProfileID string `json:"profile_id"`
}

type ListSnapshotsReply struct {
	Snapshots []model.SnapshotSummary `json:"snapshots"`
}

type ListSnapshotsResult struct {
	Reply *ListSnapshotsReply `json:"reply,omitempty"`
	Error *StructuredError    `json:"error,omitempty"`
}

type RollbackToSnapshotRequest struct {
	SnapshotID string `json:"snapshot_id"`
}

type RollbackToSnapshotReply struct{}

type RollbackToSnapshotResult struct {
	Reply *RollbackToSnapshotReply `json:"reply,omitempty"`
	Error *StructuredError         `json:"error,omitempty"`
}
