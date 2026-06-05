package qkboxd

import (
	"context"
	"database/sql"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
	"sync"
	"time"
)

type RuntimeService struct {
	*ContentCodec
	db       *persistence.DB
	engine   *EngineController
	events   *RuntimeEventHub
	platform *PlatformService
	opMu     *sync.Mutex
}

func (s *RuntimeService) EngineStart(ctx context.Context, _ api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if sErr := s.engine.Start(func() (RuntimeStartTarget, *api.StructuredError) {
		return s.loadActiveRuntimeStartTarget(ctx)
	}); sErr != nil {
		return api.EngineStartReply{}, sErr
	}
	return api.EngineStartReply{}, nil
}

func (s *RuntimeService) loadActiveRuntimeStartTarget(ctx context.Context) (RuntimeStartTarget, *api.StructuredError) {
	activeSnapshotID, err := s.db.GetActiveSnapshotID()
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalError(err)
	}
	if activeSnapshotID == "" {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "No active snapshot to start.", "qkboxd", true)
	}

	return s.loadPreparedRuntimeStartTarget(ctx, activeSnapshotID)
}

func (s *RuntimeService) loadPreparedRuntimeStartTarget(ctx context.Context, snapshotID string) (RuntimeStartTarget, *api.StructuredError) {
	target, structured := s.loadRuntimeStartTargetByID(snapshotID)
	if structured != nil {
		return RuntimeStartTarget{}, structured
	}
	if structured := s.platform.prepareRuntimeStartTargetCapabilities(ctx, target); structured != nil {
		return RuntimeStartTarget{}, structured
	}
	return target, nil
}

func (s *RuntimeService) startPreparedSnapshotTarget(ctx context.Context, snapshotID string) *api.StructuredError {
	return s.engine.Start(func() (RuntimeStartTarget, *api.StructuredError) {
		return s.loadPreparedRuntimeStartTarget(ctx, snapshotID)
	})
}

func (s *RuntimeService) loadRuntimeStartTargetByID(snapshotID string) (RuntimeStartTarget, *api.StructuredError) {
	snapshot, contentID, err := s.db.GetSnapshot(snapshotID)
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalError(err)
	}
	if snapshot == nil {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}
	if contentID == "" {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "Snapshot has no content.", "qkboxd", true)
	}

	configJSON, err := s.decryptContent(contentID)
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalErrorMessage("Failed to decrypt snapshot content: " + err.Error())
	}

	required := snapshot.RequiredCapabilities
	if len(required) == 0 {
		required = extractRequiredCapabilities(configJSON)
	}
	return RuntimeStartTarget{
		SnapshotID:           snapshotID,
		ConfigJSON:           configJSON,
		RequiredCapabilities: append([]string(nil), required...),
	}, nil
}

func (s *RuntimeService) EngineStop(_ context.Context, _ api.EngineStopRequest) (api.EngineStopReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if sErr := s.platform.restoreProxyIfOwned(); sErr != nil {
		return api.EngineStopReply{}, sErr
	}
	if err := s.engine.Stop(); err != nil {
		return api.EngineStopReply{}, err
	}
	return api.EngineStopReply{}, nil
}

func (s *RuntimeService) EngineGetStatus(_ context.Context, _ api.EngineGetStatusRequest) (api.EngineGetStatusReply, *api.StructuredError) {
	status := s.engine.GetStatus()
	if status.State == model.EngineStateIdle || status.State == model.EngineStateUninitialized {
		activeSnapshotID, err := s.db.GetActiveSnapshotID()
		if err != nil {
			return api.EngineGetStatusReply{}, qkboxdInternalError(err)
		}
		status.ActiveSnapshotID = activeSnapshotID
	}
	return api.EngineGetStatusReply{Status: status}, nil
}

func (s *RuntimeService) EngineReload(ctx context.Context, req api.EngineReloadRequest) (api.EngineReloadReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	reply := api.EngineReloadReply{TargetSnapshotID: req.SnapshotID}
	if req.SnapshotID == "" {
		return reply, api.NewStructuredError(api.ErrorIPCInvalidRequest, "snapshot_id is required.", "qkboxd", true)
	}

	status := s.engine.GetStatus()
	if status.State != model.EngineStateStarted {
		return reply, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine reload requires a running engine.", "qkboxd", true)
	}
	previousSnapshotID := status.ActiveSnapshotID
	if previousSnapshotID == "" {
		var err error
		previousSnapshotID, err = s.db.GetActiveSnapshotID()
		if err != nil {
			return reply, qkboxdInternalError(err)
		}
	}
	reply.PreviousSnapshotID = previousSnapshotID
	reply.ActiveSnapshotID = previousSnapshotID

	target, structured := s.loadRuntimeStartTargetByID(req.SnapshotID)
	if structured != nil {
		reply.Outcome = reloadOutcomeForTargetLoadFailure(structured)
		reply.Failure = structured
		return reply, nil
	}
	diag := validateContent(target.ConfigJSON)
	if diag.Status == model.ValidationStatusInvalid {
		reply.Outcome = api.ReloadOutcomeFailedValidation
		reply.Failure = &api.StructuredError{
			Code:        api.ErrorConfigValidationFailed,
			Message:     "Validation failed. Fix errors before reloading.",
			Detail:      diag.Entries,
			Source:      "qkboxd",
			Recoverable: true,
			UserAction:  "Fix the validation errors in your profile.",
		}
		return reply, nil
	}
	if structured := s.platform.prepareRuntimeStartTargetCapabilities(ctx, target); structured != nil {
		reply.Outcome = api.ReloadOutcomeFailedPlatformPrepare
		reply.Failure = structured
		return reply, nil
	}

	if _, structured := s.loadRuntimeStartTargetByID(previousSnapshotID); structured != nil {
		reply.Outcome = api.ReloadOutcomeDegraded
		reply.Failure = structured
		return reply, nil
	}

	if cleanup := s.platform.restoreProxyIfOwned(); cleanup != nil {
		reply.Outcome = api.ReloadOutcomeCleanupFailed
		reply.CleanupFailure = cleanup
		return reply, nil
	}

	if stopErr := s.engine.Stop(); stopErr != nil {
		reply.Outcome = api.ReloadOutcomeCleanupFailed
		reply.CleanupFailure = stopErr
		return reply, nil
	}

	if startErr := s.startPreparedSnapshotTarget(ctx, target.SnapshotID); startErr != nil {
		reply.Failure = startErr
		if rollbackErr := s.startPreparedSnapshotTarget(ctx, previousSnapshotID); rollbackErr != nil {
			reply.Outcome = api.ReloadOutcomeDegraded
			reply.CleanupFailure = rollbackErr
			return reply, nil
		}
		reply.Outcome = api.ReloadOutcomeRolledBack
		reply.ActiveSnapshotID = previousSnapshotID
		return reply, nil
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveSnapshotTx(tx, target.SnapshotID)
	}); err != nil {
		reply.Failure = qkboxdInternalError(err)
		if stopErr := s.engine.Stop(); stopErr != nil {
			reply.Outcome = api.ReloadOutcomeDegraded
			reply.CleanupFailure = stopErr
			return reply, nil
		}
		if rollbackErr := s.startPreparedSnapshotTarget(ctx, previousSnapshotID); rollbackErr != nil {
			reply.Outcome = api.ReloadOutcomeDegraded
			reply.CleanupFailure = rollbackErr
			return reply, nil
		}
		reply.Outcome = api.ReloadOutcomeRolledBack
		reply.ActiveSnapshotID = previousSnapshotID
		return reply, nil
	}

	reply.Outcome = api.ReloadOutcomeSuccess
	reply.ActiveSnapshotID = target.SnapshotID
	return reply, nil
}

func reloadOutcomeForTargetLoadFailure(err *api.StructuredError) api.ReloadOutcome {
	if err == nil {
		return api.ReloadOutcomeFailedTargetLoad
	}
	switch err.Code {
	case api.ErrorSnapshotNotFound, api.ErrorEngineNoActiveSnapshot:
		return api.ReloadOutcomeFailedValidation
	default:
		return api.ReloadOutcomeFailedTargetLoad
	}
}

func (s *RuntimeService) EngineSubscribeStatus(ctx context.Context, _ api.EngineSubscribeStatusRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.events.SubscribeStatus(ctx), nil
}

func (s *RuntimeService) EngineSubscribeLogs(ctx context.Context, _ api.EngineSubscribeLogsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.events.SubscribeLogs(ctx), nil
}

func (s *RuntimeService) EngineSubscribeTraffic(ctx context.Context, _ api.EngineSubscribeTrafficRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.engine.SubscribeTraffic(ctx), nil
}

func (s *RuntimeService) EngineSubscribeConnections(ctx context.Context, _ api.EngineSubscribeConnectionsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.engine.SubscribeConnections(ctx), nil
}

func (s *RuntimeService) EngineGetRuntimeCapabilities(_ context.Context, _ api.EngineGetRuntimeCapabilitiesRequest) (api.EngineGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return api.EngineGetRuntimeCapabilitiesReply{Capabilities: s.engine.RuntimeCapabilities()}, nil
}

func (s *RuntimeService) EngineListGroups(_ context.Context, _ api.EngineListGroupsRequest) (api.EngineListGroupsReply, *api.StructuredError) {
	groups, err := s.engine.ListGroups()
	if err != nil {
		return api.EngineListGroupsReply{}, err
	}
	return api.EngineListGroupsReply{Groups: groups}, nil
}

func (s *RuntimeService) EngineSelectOutbound(_ context.Context, req api.EngineSelectOutboundRequest) (api.EngineSelectOutboundReply, *api.StructuredError) {
	group, err := s.engine.SelectOutbound(req.GroupTag, req.OutboundTag)
	if err != nil {
		return api.EngineSelectOutboundReply{}, err
	}
	return api.EngineSelectOutboundReply{Group: group}, nil
}

func (s *RuntimeService) EngineURLTest(ctx context.Context, req api.EngineURLTestRequest) (api.EngineURLTestReply, *api.StructuredError) {
	timeout := 10 * time.Second
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
	}
	results, err := s.engine.URLTest(ctx, req.GroupTag, timeout)
	if err != nil {
		return api.EngineURLTestReply{}, err
	}
	return api.EngineURLTestReply{Results: results}, nil
}

func (s *RuntimeService) EngineCloseConnection(_ context.Context, req api.EngineCloseConnectionRequest) (api.EngineCloseConnectionReply, *api.StructuredError) {
	if err := s.engine.CloseConnection(req.ConnectionID); err != nil {
		return api.EngineCloseConnectionReply{}, err
	}
	return api.EngineCloseConnectionReply{}, nil
}

func (s *RuntimeService) EngineCloseAllConnections(_ context.Context, _ api.EngineCloseAllConnectionsRequest) (api.EngineCloseAllConnectionsReply, *api.StructuredError) {
	if err := s.engine.CloseAllConnections(); err != nil {
		return api.EngineCloseAllConnectionsReply{}, err
	}
	return api.EngineCloseAllConnectionsReply{}, nil
}
