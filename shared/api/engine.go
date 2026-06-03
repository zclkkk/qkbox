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
