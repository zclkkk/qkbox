package api

import "github.com/zclkkk/qkbox/shared/model"

type EngineStatus struct {
	State            model.EngineState `json:"state"`
	ActiveSnapshotID string            `json:"active_snapshot_id,omitempty"`
	StartedAt        int64             `json:"started_at,omitempty"`
	LastErrorCode    string            `json:"last_error_code,omitempty"`
	LastErrorMessage string            `json:"last_error_message,omitempty"`
}

type EngineStartRequest struct{}
type EngineStartReply struct{}
type EngineStartResult struct {
	Reply *EngineStartReply `json:"reply,omitempty"`
	Error *StructuredError  `json:"error,omitempty"`
}

type EngineStopRequest struct{}
type EngineStopReply struct{}
type EngineStopResult struct {
	Reply *EngineStopReply `json:"reply,omitempty"`
	Error *StructuredError `json:"error,omitempty"`
}

type EngineGetStatusRequest struct{}
type EngineGetStatusReply struct {
	Status EngineStatus `json:"status"`
}
type EngineGetStatusResult struct {
	Reply *EngineGetStatusReply `json:"reply,omitempty"`
	Error *StructuredError      `json:"error,omitempty"`
}

type ReloadOutcome string

const (
	ReloadOutcomeSuccess               ReloadOutcome = "success"
	ReloadOutcomeFailedTargetLoad      ReloadOutcome = "failed_target_load"
	ReloadOutcomeFailedValidation      ReloadOutcome = "failed_validation"
	ReloadOutcomeFailedPermission      ReloadOutcome = "failed_permission"
	ReloadOutcomeFailedPlatformPrepare ReloadOutcome = "failed_platform_prepare"
	ReloadOutcomeFailedRuntimeStart    ReloadOutcome = "failed_runtime_start"
	ReloadOutcomeRolledBack            ReloadOutcome = "rolled_back"
	ReloadOutcomeDegraded              ReloadOutcome = "degraded"
	ReloadOutcomeCleanupFailed         ReloadOutcome = "cleanup_failed"
)

type EngineReloadRequest struct {
	SnapshotID string `json:"snapshot_id"`
}

type EngineReloadReply struct {
	Outcome            ReloadOutcome    `json:"outcome"`
	TargetSnapshotID   string           `json:"target_snapshot_id"`
	PreviousSnapshotID string           `json:"previous_snapshot_id,omitempty"`
	ActiveSnapshotID   string           `json:"active_snapshot_id,omitempty"`
	Failure            *StructuredError `json:"failure,omitempty"`
	CleanupFailure     *StructuredError `json:"cleanup_failure,omitempty"`
}

type EngineReloadResult struct {
	Reply *EngineReloadReply `json:"reply,omitempty"`
	Error *StructuredError   `json:"error,omitempty"`
}
