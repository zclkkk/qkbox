package qkboxd

import (
	"context"
	"database/sql"
	"encoding/json"
	"runtime"
	"time"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/redact"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type Service struct {
	db     *persistence.DB
	key    []byte
	engine *EngineController
}

func NewService(db *persistence.DB, key []byte) *Service {
	return &Service{db: db, key: key, engine: NewEngineController()}
}

// Hello (existing)

func (s *Service) Hello(_ context.Context, req api.HelloRequest) (api.HelloReply, *api.StructuredError) {
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
		RuntimeCapabilities:  api.RuntimeCapabilityShell(),
		PlatformCapabilities: api.PlatformCapabilityShell(),
	}, nil
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

	snapshot := model.Snapshot{
		ID:               persistence.NewSnapshotID(),
		ProfileID:        req.ProfileID,
		ValidationStatus: diag.Status,
		Diagnostics:      diag.Entries,
		CreatedAt:        time.Now().UnixMilli(),
	}
	snapshotContent, err := s.encryptedContent("snapshot", req.ProfileID, content, snapshot.CreatedAt)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateSnapshotWithContentTx(tx, &snapshot, snapshotContent, diagJSON, nil, nil)
	}); err != nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}

	return api.CreateProfileSnapshotReply{Snapshot: snapshot}, nil
}

func (s *Service) ActivateProfileSnapshot(_ context.Context, req api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError) {
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

func (s *Service) EngineStart(_ context.Context, _ api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError) {
	activeSnapshotID, err := s.db.GetActiveSnapshotID()
	if err != nil {
		return api.EngineStartReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if activeSnapshotID == "" {
		return api.EngineStartReply{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "No active snapshot to start.", "qkboxd", true)
	}

	_, contentID, err := s.db.GetSnapshot(activeSnapshotID)
	if err != nil {
		return api.EngineStartReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
	}
	if contentID == "" {
		return api.EngineStartReply{}, api.NewStructuredError(api.ErrorEngineNoActiveSnapshot, "Active snapshot has no content.", "qkboxd", true)
	}

	configJSON, err := s.decryptContent(contentID)
	if err != nil {
		return api.EngineStartReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "Failed to decrypt snapshot content: "+err.Error(), "qkboxd", false)
	}

	if sErr := s.engine.Start(configJSON, activeSnapshotID); sErr != nil {
		return api.EngineStartReply{}, sErr
	}

	return api.EngineStartReply{}, nil
}

func (s *Service) EngineStop(_ context.Context, _ api.EngineStopRequest) (api.EngineStopReply, *api.StructuredError) {
	if err := s.engine.Stop(); err != nil {
		return api.EngineStopReply{}, err
	}
	return api.EngineStopReply{}, nil
}

func (s *Service) EngineGetStatus(_ context.Context, _ api.EngineGetStatusRequest) (api.EngineGetStatusReply, *api.StructuredError) {
	return api.EngineGetStatusReply{Status: s.engine.GetStatus()}, nil
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
