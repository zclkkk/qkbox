package qkboxd

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type RuntimeService struct {
	db       *persistence.DB
	engine   *EngineController
	events   *RuntimeEventHub
	platform *PlatformService
	opMu     *sync.Mutex
}

func (s *RuntimeService) EngineStart(ctx context.Context, _ api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	target, structured := s.loadActiveRuntimeStartTarget(ctx)
	if structured != nil {
		return api.EngineStartReply{}, structured
	}
	if sErr := s.engine.Start(target); sErr != nil {
		return api.EngineStartReply{}, sErr
	}
	return api.EngineStartReply{}, nil
}

func (s *RuntimeService) loadActiveRuntimeStartTarget(ctx context.Context) (RuntimeStartTarget, *api.StructuredError) {
	activeProfileID, err := s.db.GetActiveProfileID()
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalError(err)
	}
	if activeProfileID == "" {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorEngineNoActiveProfile, "No active profile to start.", "qkboxd", true)
	}

	return s.loadPreparedRuntimeStartTarget(ctx, activeProfileID)
}

func (s *RuntimeService) loadPreparedRuntimeStartTarget(ctx context.Context, profileID string) (RuntimeStartTarget, *api.StructuredError) {
	target, structured := s.loadRuntimeStartTargetByID(profileID)
	if structured != nil {
		return RuntimeStartTarget{}, structured
	}
	if structured := s.platform.prepareRuntimeStartTargetCapabilities(ctx, target); structured != nil {
		return RuntimeStartTarget{}, structured
	}
	return target, nil
}

func (s *RuntimeService) loadRuntimeStartTargetByID(profileID string) (RuntimeStartTarget, *api.StructuredError) {
	profile, err := s.db.GetProfile(profileID)
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}
	configJSON, err := s.db.GetProfileContent(profileID)
	if err != nil {
		return RuntimeStartTarget{}, qkboxdInternalError(err)
	}
	if configJSON == "" {
		return RuntimeStartTarget{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is empty.", "qkboxd", true)
	}
	if diag := validateProfileConfig(profileID, configJSON); diag.Status == model.ValidationStatusInvalid {
		return RuntimeStartTarget{}, profileConfigValidationError(
			"Profile content failed validation.",
			diag,
			"Fix the profile content before activating it.",
		)
	}
	return RuntimeStartTarget{
		ProfileID:            profileID,
		ConfigJSON:           configJSON,
		RequiredCapabilities: extractRequiredCapabilities(configJSON),
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
		activeProfileID, err := s.db.GetActiveProfileID()
		if err != nil {
			return api.EngineGetStatusReply{}, qkboxdInternalError(err)
		}
		status.ActiveProfileID = activeProfileID
	}
	return api.EngineGetStatusReply{Status: status}, nil
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
