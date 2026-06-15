package qkboxd

import (
	"context"
	"database/sql"

	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type activationRollback struct {
	oldProfileID         string
	oldRuntimeWasRunning bool
	proxyRecord          *proxyOwnerRecord
}

func (s *Service) ActivateProfile(ctx context.Context, req api.ActivateProfileRequest) (api.ActivateProfileReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.ActivateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}

	s.RuntimeService.opMu.Lock()
	defer s.RuntimeService.opMu.Unlock()

	oldProfileID, err := s.db.GetActiveProfileID()
	if err != nil {
		return api.ActivateProfileReply{}, qkboxdInternalError(err)
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.ActivateProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.ActivateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	target, structured := s.RuntimeService.loadRuntimeStartTargetByID(req.ProfileID)
	if structured != nil {
		return api.ActivateProfileReply{}, structured
	}

	status, structured := s.engine.CheckActivationAllowed()
	if structured != nil {
		return api.ActivateProfileReply{}, structured
	}

	proxyRecord, structured := s.PlatformService.captureOwnedProxyForActivation()
	if structured != nil {
		return api.ActivateProfileReply{}, structured
	}
	if proxyRecord != nil {
		if structured := s.PlatformService.restoreCapturedProxy(proxyRecord); structured != nil {
			return api.ActivateProfileReply{}, structured
		}
	}

	oldRuntimeWasRunning := status.State == model.EngineStateStarted
	if oldRuntimeWasRunning {
		if structured := s.engine.Stop(); structured != nil {
			return api.ActivateProfileReply{}, structured
		}
	}

	rollback := activationRollback{
		oldProfileID:         oldProfileID,
		oldRuntimeWasRunning: oldRuntimeWasRunning,
		proxyRecord:          proxyRecord,
	}

	if structured := s.PlatformService.prepareRuntimeStartTargetCapabilities(ctx, target); structured != nil {
		return api.ActivateProfileReply{}, s.rollbackActivation(ctx, rollback, false, false, structured)
	}
	if structured := s.engine.Start(target); structured != nil {
		return api.ActivateProfileReply{}, s.rollbackActivation(ctx, rollback, false, false, structured)
	}

	proxyRebound := false
	if proxyRecord != nil {
		if structured := s.PlatformService.bindCapturedProxyToRuntime(proxyRecord); structured != nil {
			return api.ActivateProfileReply{}, s.rollbackActivation(ctx, rollback, true, proxyRebound, structured)
		}
		proxyRebound = true
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveProfileTx(tx, req.ProfileID)
	}); err != nil {
		return api.ActivateProfileReply{}, s.rollbackActivation(ctx, rollback, true, proxyRebound, qkboxdInternalError(err))
	}

	return api.ActivateProfileReply{Profile: *profile}, nil
}

func (s *Service) rollbackActivation(ctx context.Context, rollback activationRollback, newRuntimeStarted bool, proxyRebound bool, original *api.StructuredError) *api.StructuredError {
	if proxyRebound {
		if structured := s.PlatformService.restoreCapturedProxy(rollback.proxyRecord); structured != nil {
			return attachRollbackError(original, "restore_proxy", structured)
		}
	}
	if newRuntimeStarted {
		if structured := s.engine.Stop(); structured != nil {
			return attachRollbackError(original, "stop_new_runtime", structured)
		}
	}
	if rollback.oldRuntimeWasRunning && rollback.oldProfileID != "" {
		target, structured := s.RuntimeService.loadRuntimeStartTargetByID(rollback.oldProfileID)
		if structured != nil {
			return attachRollbackError(original, "load_old_profile", structured)
		}
		if structured := s.PlatformService.prepareRuntimeStartTargetCapabilities(ctx, target); structured != nil {
			return attachRollbackError(original, "prepare_old_profile", structured)
		}
		if structured := s.engine.Start(target); structured != nil {
			return attachRollbackError(original, "start_old_runtime", structured)
		}
	}
	return original
}

func attachRollbackError(original *api.StructuredError, stage string, rollback *api.StructuredError) *api.StructuredError {
	if original == nil || rollback == nil {
		return original
	}
	original.Detail = map[string]interface{}{
		"original_detail": original.Detail,
		"rollback_stage":  stage,
		"rollback_error":  rollback,
	}
	return original
}
