package qkboxd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/redact"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type Service struct {
	db         *persistence.DB
	key        []byte
	engine     *EngineController
	events     *RuntimeEventHub
	proxy      capability.SystemProxyProvider
	privileged capability.PrivilegedProvider
	opMu       sync.Mutex
}

const privilegedCapabilityProbeTimeout = 500 * time.Millisecond

func NewService(runtimeCtx context.Context, db *persistence.DB, key []byte, proxy capability.SystemProxyProvider, privileged capability.PrivilegedProvider) *Service {
	events := NewRuntimeEventHub()
	return &Service{
		db:         db,
		key:        key,
		events:     events,
		engine:     NewEngineController(runtimeCtx, events),
		proxy:      proxy,
		privileged: privileged,
	}
}

func (s *Service) Close() error {
	s.bestEffortProxyRestore()
	return s.engine.Shutdown()
}

// Hello

func (s *Service) Hello(ctx context.Context, req api.HelloRequest) (api.HelloReply, *api.StructuredError) {
	if req.ClientAPIVersion != api.APIVersion {
		return api.HelloReply{}, api.VersionUnsupported(req.ClientAPIVersion)
	}
	return api.HelloReply{
		APIVersion:             api.APIVersion,
		MinSupportedAPIVersion: api.MinSupportedAPIVersion,
		SchemaRevision:         api.SchemaRevision,
		AppVersion:             api.AppVersion,
		QKBoxDVersion:          api.QKBoxDVersion,
		Platform: model.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		RuntimeCapabilities:  s.engine.RuntimeCapabilities(),
		PlatformCapabilities: s.platformCapabilities(ctx),
	}, nil
}

func (s *Service) platformCapabilities(ctx context.Context) []api.Capability {
	caps := api.PlatformCapabilityShell()
	if s.proxy == nil {
		return s.applyPrivilegedCapabilities(ctx, caps)
	}
	avail := s.proxy.Availability()
	for i, cap := range caps {
		if cap.Name == api.CapabilitySystemProxy {
			if avail.Available && avail.Supported {
				caps[i].State = api.CapabilityAvailable
				caps[i].Reason = ""
			} else if avail.Available && !avail.Supported {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = avail.Reason
			} else {
				caps[i].State = api.CapabilityUnsupported
				caps[i].Reason = avail.Reason
			}
			break
		}
	}
	return s.applyPrivilegedCapabilities(ctx, caps)
}

func (s *Service) applyPrivilegedCapabilities(ctx context.Context, caps []api.Capability) []api.Capability {
	status := api.PrivilegedProviderStatus{Reason: "Privileged provider is not configured."}
	if s.privileged != nil {
		probeCtx, cancel := context.WithTimeout(ctx, privilegedCapabilityProbeTimeout)
		defer cancel()
		status = s.privileged.Status(probeCtx)
	}
	providerReady := status.Installed && status.Reachable && status.Authenticated && status.Version == status.ExpectedVersion
	for i, cap := range caps {
		switch cap.Name {
		case api.CapabilityBackgroundService:
			if providerReady {
				caps[i].State = api.CapabilityAvailable
				caps[i].Reason = ""
			} else {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = providerStatusReason(status)
			}
		case api.CapabilityTunMode, api.CapabilityDNSHijack:
			if providerReady {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = "Privileged network mutation is not available in this provider build."
			} else {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = providerStatusReason(status)
			}
		}
	}
	return caps
}

func providerStatusReason(status api.PrivilegedProviderStatus) string {
	if status.Reason != "" {
		return status.Reason
	}
	if !status.Installed {
		return "Privileged provider is not installed."
	}
	if !status.Reachable {
		return "Privileged provider is not reachable."
	}
	if !status.Authenticated {
		return "Privileged provider authentication failed."
	}
	if status.ExpectedVersion != "" && status.Version != status.ExpectedVersion {
		return "Privileged provider version mismatch."
	}
	return "Privileged provider is unavailable."
}

// Profile CRUD

func (s *Service) CreateProfile(_ context.Context, req api.CreateProfileRequest) (api.CreateProfileReply, *api.StructuredError) {
	if req.Name == "" {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorProfileNameEmpty, "Profile name is required.", "qkboxd", true)
	}
	if req.Content == "" {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is required.", "qkboxd", true)
	}

	now := time.Now().UnixMilli()
	profile := model.Profile{
		ID:        persistence.NewProfileID(),
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	draftContent, err := s.encryptedContent("draft", profile.ID, req.Content, now)
	if err != nil {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateProfileWithDraftTx(tx, &profile, draftContent)
	}); err != nil {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.CreateProfileReply{Profile: profile}, nil
}

func (s *Service) UpdateProfileDraft(_ context.Context, req api.UpdateProfileDraftRequest) (api.UpdateProfileDraftReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}
	if req.Content == "" {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is required.", "qkboxd", true)
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	draftContent, err := s.encryptedContent("draft", req.ProfileID, req.Content, time.Now().UnixMilli())
	if err != nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.ReplaceDraftContentTx(tx, req.ProfileID, draftContent)
	}); err != nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	profile, err = s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile disappeared after update.", "qkboxd", false)
	}
	return api.UpdateProfileDraftReply{Profile: *profile}, nil
}

func (s *Service) DeleteProfile(_ context.Context, req api.DeleteProfileRequest) (api.DeleteProfileReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}
	if profile.ActiveSnapshotID != nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorProfileHasSnapshot, "Deactivate the active snapshot before deleting.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.DeleteProfileGraphTx(tx, req.ProfileID)
	}); err != nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.DeleteProfileReply{}, nil
}

func (s *Service) ListProfiles(_ context.Context, _ api.ListProfilesRequest) (api.ListProfilesReply, *api.StructuredError) {
	profiles, err := s.db.ListProfiles()
	if err != nil {
		return api.ListProfilesReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profiles == nil {
		profiles = []model.ProfileSummary{}
	}
	return api.ListProfilesReply{Profiles: profiles}, nil
}

func (s *Service) GetProfile(_ context.Context, req api.GetProfileRequest) (api.GetProfileReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.GetProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.GetProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	reply := api.GetProfileReply{Profile: *profile}
	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.GetProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if contentID != "" {
		content, err := s.decryptContent(contentID)
		if err != nil {
			return api.GetProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt draft content: "+err.Error(), "qkboxd", false)
		}
		reply.Content = content
	}
	return reply, nil
}

// Snapshot lifecycle

func (s *Service) ValidateProfileDraft(_ context.Context, req api.ValidateProfileDraftRequest) (api.ValidateProfileDraftReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.ValidateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.ValidateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.ValidateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if contentID == "" {
		return api.ValidateProfileDraftReply{Diagnostics: model.Diagnostics{
			ProfileID: req.ProfileID,
			Status:    model.ValidationStatusInvalid,
			Entries: []model.ValidationDiagnostic{{
				Severity: model.SeverityError,
				Message:  "Profile has no draft content.",
			}},
		}}, nil
	}

	content, err := s.decryptContent(contentID)
	if err != nil {
		return api.ValidateProfileDraftReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt content: "+err.Error(), "qkboxd", false)
	}

	diag := validateContent(content)
	diag.ProfileID = req.ProfileID
	diag.RedactedPreview = redact.Content(content)
	return api.ValidateProfileDraftReply{Diagnostics: diag}, nil
}

func (s *Service) GetProfileDiagnostics(_ context.Context, req api.GetProfileDiagnosticsRequest) (api.GetProfileDiagnosticsReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.GetProfileDiagnosticsReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if contentID == "" {
		return api.GetProfileDiagnosticsReply{Diagnostics: model.Diagnostics{
			ProfileID: req.ProfileID,
			Status:    model.ValidationStatusUnknown,
		}}, nil
	}

	content, err := s.decryptContent(contentID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt content: "+err.Error(), "qkboxd", false)
	}

	diag := validateContent(content)
	diag.ProfileID = req.ProfileID
	diag.RedactedPreview = redact.Content(content)
	return api.GetProfileDiagnosticsReply{Diagnostics: diag}, nil
}

func (s *Service) CreateProfileSnapshot(_ context.Context, req api.CreateProfileSnapshotRequest) (api.CreateProfileSnapshotReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if profile == nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if contentID == "" {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotCreateFailed, "Profile has no draft content.", "qkboxd", true)
	}

	content, err := s.decryptContent(contentID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt content: "+err.Error(), "qkboxd", false)
	}

	diag := validateContent(content)
	if diag.Status == model.ValidationStatusInvalid {
		return api.CreateProfileSnapshotReply{}, &api.StructuredError{
			Code:        api.ErrorConfigValidationFailed,
			Message:     "Validation failed. Fix errors before creating a snapshot.",
			Detail:      diag.Entries,
			Source:      "qkboxd",
			Recoverable: true,
			UserAction:  "Fix the validation errors in your profile.",
		}
	}

	diagJSON, err := json.Marshal(diag.Entries)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	requiredCapabilities := extractRequiredCapabilities(content)
	requiredCapabilitiesJSON, err := json.Marshal(requiredCapabilities)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	snapshot := model.Snapshot{
		ID:                   persistence.NewSnapshotID(),
		ProfileID:            req.ProfileID,
		ValidationStatus:     diag.Status,
		Diagnostics:          diag.Entries,
		RequiredCapabilities: requiredCapabilities,
		CreatedAt:            time.Now().UnixMilli(),
	}
	snapshotContent, err := s.encryptedContent("snapshot", req.ProfileID, content, snapshot.CreatedAt)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateSnapshotWithContentTx(tx, &snapshot, snapshotContent, diagJSON, nil, requiredCapabilitiesJSON)
	}); err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.CreateProfileSnapshotReply{Snapshot: snapshot}, nil
}

func (s *Service) ActivateProfileSnapshot(_ context.Context, req api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if err := s.engine.CheckBlockMutations(); err != nil {
		return api.ActivateProfileSnapshotReply{}, err
	}

	snapshot, _, err := s.db.GetSnapshot(req.SnapshotID)
	if err != nil {
		return api.ActivateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if snapshot == nil {
		return api.ActivateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveSnapshotTx(tx, snapshot.ID)
	}); err != nil {
		return api.ActivateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.ActivateProfileSnapshotReply{}, nil
}

func (s *Service) GetActiveProfile(_ context.Context, _ api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError) {
	profile, err := s.db.GetActiveProfile()
	if err != nil {
		return api.GetActiveProfileReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	return api.GetActiveProfileReply{Profile: profile}, nil
}

func (s *Service) GetActiveSnapshot(_ context.Context, _ api.GetActiveSnapshotRequest) (api.GetActiveSnapshotReply, *api.StructuredError) {
	snapshot, _, err := s.db.GetActiveSnapshot()
	if err != nil {
		return api.GetActiveSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	return api.GetActiveSnapshotReply{Snapshot: snapshot}, nil
}

func (s *Service) ListSnapshots(_ context.Context, req api.ListSnapshotsRequest) (api.ListSnapshotsReply, *api.StructuredError) {
	snapshots, err := s.db.ListSnapshots(req.ProfileID)
	if err != nil {
		return api.ListSnapshotsReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if snapshots == nil {
		snapshots = []model.SnapshotSummary{}
	}
	return api.ListSnapshotsReply{Snapshots: snapshots}, nil
}

func (s *Service) RollbackToSnapshot(_ context.Context, req api.RollbackToSnapshotRequest) (api.RollbackToSnapshotReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if err := s.engine.CheckBlockMutations(); err != nil {
		return api.RollbackToSnapshotReply{}, err
	}

	snapshot, _, err := s.db.GetSnapshot(req.SnapshotID)
	if err != nil {
		return api.RollbackToSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if snapshot == nil {
		return api.RollbackToSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveSnapshotTx(tx, snapshot.ID)
	}); err != nil {
		return api.RollbackToSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.RollbackToSnapshotReply{}, nil
}

// Engine lifecycle

func (s *Service) EngineStart(ctx context.Context, _ api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if sErr := s.engine.Start(func() (EngineStartTarget, *api.StructuredError) {
		return s.loadActiveEngineStartTarget(ctx)
	}); sErr != nil {
		return api.EngineStartReply{}, sErr
	}
	return api.EngineStartReply{}, nil
}

func (s *Service) loadActiveEngineStartTarget(ctx context.Context) (EngineStartTarget, *api.StructuredError) {
	activeSnapshotID, err := s.db.GetActiveSnapshotID()
	if err != nil {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if activeSnapshotID == "" {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "No active snapshot to start.", "qkboxd", true)
	}

	return s.loadPreparedEngineStartTarget(ctx, activeSnapshotID)
}

func (s *Service) loadPreparedEngineStartTarget(ctx context.Context, snapshotID string) (EngineStartTarget, *api.StructuredError) {
	target, structured := s.loadEngineStartTargetByID(snapshotID)
	if structured != nil {
		return EngineStartTarget{}, structured
	}
	if structured := s.prepareSnapshotCapabilities(ctx, target.SnapshotID, target.ConfigJSON); structured != nil {
		return EngineStartTarget{}, structured
	}
	return target, nil
}

func (s *Service) startPreparedSnapshotTarget(ctx context.Context, snapshotID string) *api.StructuredError {
	return s.engine.Start(func() (EngineStartTarget, *api.StructuredError) {
		return s.loadPreparedEngineStartTarget(ctx, snapshotID)
	})
}

func (s *Service) loadEngineStartTargetByID(snapshotID string) (EngineStartTarget, *api.StructuredError) {
	snapshot, contentID, err := s.db.GetSnapshot(snapshotID)
	if err != nil {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if snapshot == nil {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}
	if contentID == "" {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "Snapshot has no content.", "qkboxd", true)
	}

	configJSON, err := s.decryptContent(contentID)
	if err != nil {
		return EngineStartTarget{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt snapshot content: "+err.Error(), "qkboxd", false)
	}

	return EngineStartTarget{SnapshotID: snapshotID, ConfigJSON: configJSON}, nil
}

func (s *Service) EngineStop(_ context.Context, _ api.EngineStopRequest) (api.EngineStopReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if sErr := s.restoreProxyIfOwned(); sErr != nil {
		return api.EngineStopReply{}, sErr
	}
	if err := s.engine.Stop(); err != nil {
		return api.EngineStopReply{}, err
	}
	return api.EngineStopReply{}, nil
}

func (s *Service) EngineGetStatus(_ context.Context, _ api.EngineGetStatusRequest) (api.EngineGetStatusReply, *api.StructuredError) {
	status := s.engine.GetStatus()
	if status.State == model.EngineStateIdle || status.State == model.EngineStateUninitialized {
		activeSnapshotID, err := s.db.GetActiveSnapshotID()
		if err != nil {
			return api.EngineGetStatusReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
		}
		status.ActiveSnapshotID = activeSnapshotID
	}
	return api.EngineGetStatusReply{Status: status}, nil
}

func (s *Service) EngineReload(ctx context.Context, req api.EngineReloadRequest) (api.EngineReloadReply, *api.StructuredError) {
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
			return reply, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
		}
	}
	reply.PreviousSnapshotID = previousSnapshotID
	reply.ActiveSnapshotID = previousSnapshotID

	target, structured := s.loadEngineStartTargetByID(req.SnapshotID)
	if structured != nil {
		reply.Outcome = api.ReloadOutcomeFailedRuntimeStart
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
	if structured := s.prepareSnapshotCapabilities(ctx, target.SnapshotID, target.ConfigJSON); structured != nil {
		reply.Outcome = api.ReloadOutcomeFailedPlatformPrepare
		reply.Failure = structured
		return reply, nil
	}

	if _, structured := s.loadEngineStartTargetByID(previousSnapshotID); structured != nil {
		reply.Outcome = api.ReloadOutcomeDegraded
		reply.Failure = structured
		return reply, nil
	}

	if cleanup := s.restoreProxyIfOwned(); cleanup != nil {
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
		reply.Failure = api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
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

func (s *Service) EngineSubscribeStatus(ctx context.Context, _ api.EngineSubscribeStatusRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.events.SubscribeStatus(ctx), nil
}

func (s *Service) EngineSubscribeLogs(ctx context.Context, _ api.EngineSubscribeLogsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.events.SubscribeLogs(ctx), nil
}

func (s *Service) EngineSubscribeTraffic(ctx context.Context, _ api.EngineSubscribeTrafficRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.engine.SubscribeTraffic(ctx), nil
}

func (s *Service) EngineSubscribeConnections(ctx context.Context, _ api.EngineSubscribeConnectionsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return s.engine.SubscribeConnections(ctx), nil
}

func (s *Service) EngineGetRuntimeCapabilities(_ context.Context, _ api.EngineGetRuntimeCapabilitiesRequest) (api.EngineGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return api.EngineGetRuntimeCapabilitiesReply{Capabilities: s.engine.RuntimeCapabilities()}, nil
}

func (s *Service) EngineListGroups(_ context.Context, _ api.EngineListGroupsRequest) (api.EngineListGroupsReply, *api.StructuredError) {
	groups, err := s.engine.ListGroups()
	if err != nil {
		return api.EngineListGroupsReply{}, err
	}
	return api.EngineListGroupsReply{Groups: groups}, nil
}

func (s *Service) EngineSelectOutbound(_ context.Context, req api.EngineSelectOutboundRequest) (api.EngineSelectOutboundReply, *api.StructuredError) {
	group, err := s.engine.SelectOutbound(req.GroupTag, req.OutboundTag)
	if err != nil {
		return api.EngineSelectOutboundReply{}, err
	}
	return api.EngineSelectOutboundReply{Group: group}, nil
}

func (s *Service) EngineURLTest(ctx context.Context, req api.EngineURLTestRequest) (api.EngineURLTestReply, *api.StructuredError) {
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

func (s *Service) EngineCloseConnection(_ context.Context, req api.EngineCloseConnectionRequest) (api.EngineCloseConnectionReply, *api.StructuredError) {
	if err := s.engine.CloseConnection(req.ConnectionID); err != nil {
		return api.EngineCloseConnectionReply{}, err
	}
	return api.EngineCloseConnectionReply{}, nil
}

func (s *Service) EngineCloseAllConnections(_ context.Context, _ api.EngineCloseAllConnectionsRequest) (api.EngineCloseAllConnectionsReply, *api.StructuredError) {
	if err := s.engine.CloseAllConnections(); err != nil {
		return api.EngineCloseAllConnectionsReply{}, err
	}
	return api.EngineCloseAllConnectionsReply{}, nil
}

// helpers

func (s *Service) encryptedContent(sourceType, sourceID, content string, createdAt int64) (*persistence.EncryptedContent, error) {
	iv, ct, err := qkboxcrypto.Encrypt(s.key, []byte(content))
	if err != nil {
		return nil, err
	}
	return &persistence.EncryptedContent{
		ID:         persistence.NewContentID(),
		SourceType: sourceType,
		SourceID:   sourceID,
		IV:         iv,
		Ciphertext: ct,
		CreatedAt:  createdAt,
	}, nil
}

func (s *Service) decryptContent(contentID string) (string, error) {
	content, err := s.db.GetContent(contentID)
	if err != nil {
		return "", err
	}
	if content == nil {
		return "", nil
	}
	plaintext, err := qkboxcrypto.Decrypt(s.key, content.IV, content.Ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Service) prepareSnapshotCapabilities(ctx context.Context, snapshotID, configJSON string) *api.StructuredError {
	snapshot, _, err := s.db.GetSnapshot(snapshotID)
	if err != nil {
		return api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	required := []string{}
	if snapshot != nil && len(snapshot.RequiredCapabilities) > 0 {
		required = snapshot.RequiredCapabilities
	} else {
		required = extractRequiredCapabilities(configJSON)
	}
	for _, feature := range required {
		if !isPrivilegedFeature(feature) {
			continue
		}
		if s.privileged == nil {
			return api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
		}
		reply, structured := s.privileged.PrepareFeature(ctx, feature)
		if structured != nil {
			return structured
		}
		if reply.State != api.CapabilityAvailable {
			err := api.NewStructuredError(api.ErrorPlatformPrepareFailed, "Required platform capability is not available.", "provider", true)
			err.Detail = reply
			return err
		}
	}
	return nil
}

func isPrivilegedFeature(feature string) bool {
	switch feature {
	case api.CapabilityTunMode, api.CapabilityDNSHijack, api.CapabilityBackgroundService:
		return true
	default:
		return false
	}
}

// Platform capabilities

func (s *Service) PlatformGetCapabilities(ctx context.Context, _ api.GetPlatformCapabilitiesRequest) (api.GetPlatformCapabilitiesReply, *api.StructuredError) {
	return api.GetPlatformCapabilitiesReply{Capabilities: s.platformCapabilities(ctx)}, nil
}

func (s *Service) PlatformGetPrivilegedProviderStatus(ctx context.Context, _ api.GetPrivilegedProviderStatusRequest) (api.GetPrivilegedProviderStatusReply, *api.StructuredError) {
	if s.privileged == nil {
		return api.GetPrivilegedProviderStatusReply{
			Status: api.PrivilegedProviderStatus{Reason: "Privileged provider is not configured."},
		}, nil
	}
	return api.GetPrivilegedProviderStatusReply{Status: s.privileged.Status(ctx)}, nil
}

func (s *Service) PlatformPrepareFeature(ctx context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	if !isPrivilegedFeature(req.Feature) {
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Feature is not supported by the privileged provider.", "qkboxd", true)
	}
	if s.privileged == nil {
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
	}
	return s.privileged.PrepareFeature(ctx, req.Feature)
}

func (s *Service) PlatformRunRepairAction(ctx context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	if s.privileged == nil {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
	}
	return s.privileged.RunRepairAction(ctx, req.Action)
}

func (s *Service) PlatformGetSystemProxyStatus(_ context.Context, _ api.GetSystemProxyStatusRequest) (api.GetSystemProxyStatusReply, *api.StructuredError) {
	reply := api.GetSystemProxyStatusReply{}

	if s.proxy == nil {
		return reply, nil
	}
	avail := s.proxy.Availability()
	reply.Available = avail.Available
	reply.Supported = avail.Supported
	reply.Reason = avail.Reason
	if !avail.Available || !avail.Supported {
		return reply, nil
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return reply, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	reply.OSEnabled = state.Enabled
	reply.Address = state.Addr
	reply.Port = state.Port

	record, err := loadProxyOwner(s.db)
	if err != nil {
		return reply, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record != nil && record.QKBoxOwned {
		if proxyOwnerMatches(state, record) {
			reply.QKBoxOwned = true
			reply.Address = record.ProxyAddr
			reply.Port = record.ProxyPort
		}
	}

	return reply, nil
}

func (s *Service) PlatformSetSystemProxyEnabled(_ context.Context, req api.SetSystemProxyEnabledRequest) (api.SetSystemProxyEnabledReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if s.proxy == nil || !s.proxy.Availability().Available || !s.proxy.Availability().Supported {
		return api.SetSystemProxyEnabledReply{}, api.NewStructuredError(api.ErrorPlatformProxyUnsupported, "System proxy is not available on this platform.", "platform", false)
	}

	if !req.Enabled {
		return api.SetSystemProxyEnabledReply{}, s.disableProxy()
	}

	return api.SetSystemProxyEnabledReply{}, s.enableProxy()
}

func (s *Service) enableProxy() *api.StructuredError {
	listeners, sErr := s.engine.ListenerInfo()
	if sErr != nil {
		return sErr
	}
	if len(listeners) == 0 {
		return api.NewStructuredError(api.ErrorPlatformProxyNoListener, "No HTTP/mixed inbound found in active config.", "qkboxd", true)
	}
	target := listeners[0]
	addr := target.Address
	port := target.Port

	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}

	if record != nil && record.QKBoxOwned {
		state, err := s.proxy.CurrentState()
		if err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
		}
		if proxyOwnerMatches(state, record) {
			return nil
		}
		record.ProxyAddr = addr
		record.ProxyPort = port
		record.EnabledAt = time.Now().UnixMilli()
		if err := saveProxyOwner(s.db, record); err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
		}
		if err := s.proxy.Apply(addr, port); err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
		}
		return nil
	}

	snapshot, err := s.proxy.Snapshot()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}

	newRecord := &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   snapshot,
		ProxyAddr:  addr,
		ProxyPort:  port,
		EnabledAt:  time.Now().UnixMilli(),
	}
	if err := saveProxyOwner(s.db, newRecord); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}

	if err := s.proxy.Apply(addr, port); err != nil {
		if restoreErr := s.proxy.Restore(snapshot); restoreErr == nil {
			_ = deleteProxyOwner(s.db)
		}
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}

	return nil
}

func (s *Service) disableProxy() *api.StructuredError {
	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record == nil || !record.QKBoxOwned {
		return nil
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return nil
	}

	if err := s.proxy.Restore(record.Snapshot); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	_ = deleteProxyOwner(s.db)
	return nil
}

func (s *Service) restoreProxyIfOwned() *api.StructuredError {
	if s.proxy == nil {
		return nil
	}
	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record == nil || !record.QKBoxOwned {
		return nil
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, "System proxy owner record exists but the platform provider is unavailable.", "platform", true)
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return nil
	}

	if err := s.proxy.Restore(record.Snapshot); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	_ = deleteProxyOwner(s.db)
	return nil
}

func (s *Service) bestEffortProxyRestore() {
	if s.proxy == nil {
		return
	}
	record, err := loadProxyOwner(s.db)
	if err != nil || record == nil || !record.QKBoxOwned {
		return
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		fmt.Printf("warning: system proxy owner record kept because provider is unavailable: %s\n", avail.Reason)
		return
	}
	state, err := s.proxy.CurrentState()
	if err != nil {
		fmt.Printf("warning: failed to read system proxy on shutdown: %v\n", err)
		return
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return
	}
	if err := s.proxy.Restore(record.Snapshot); err != nil {
		fmt.Printf("warning: failed to restore system proxy on shutdown: %v\n", err)
		return
	}
	_ = deleteProxyOwner(s.db)
}
