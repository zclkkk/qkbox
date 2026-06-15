package qkboxd

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/redact"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type SnapshotService struct {
	db     *persistence.DB
	engine *EngineController
	opMu   *sync.Mutex
}

func (s *SnapshotService) ValidateProfileDraft(_ context.Context, req api.ValidateProfileDraftRequest) (api.ValidateProfileDraftReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.ValidateProfileDraftReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.ValidateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.ValidateProfileDraftReply{}, qkboxdInternalError(err)
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
		return api.ValidateProfileDraftReply{}, qkboxdInternalErrorMessage("Failed to decrypt content: " + err.Error())
	}

	diag := validateContent(content)
	diag.ProfileID = req.ProfileID
	diag.RedactedPreview = redact.Content(content)
	return api.ValidateProfileDraftReply{Diagnostics: diag}, nil
}

func (s *SnapshotService) GetProfileDiagnostics(_ context.Context, req api.GetProfileDiagnosticsRequest) (api.GetProfileDiagnosticsReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.GetProfileDiagnosticsReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, qkboxdInternalError(err)
	}
	if contentID == "" {
		return api.GetProfileDiagnosticsReply{Diagnostics: model.Diagnostics{
			ProfileID: req.ProfileID,
			Status:    model.ValidationStatusUnknown,
		}}, nil
	}

	content, err := s.decryptContent(contentID)
	if err != nil {
		return api.GetProfileDiagnosticsReply{}, qkboxdInternalErrorMessage("Failed to decrypt content: " + err.Error())
	}

	diag := validateContent(content)
	diag.ProfileID = req.ProfileID
	diag.RedactedPreview = redact.Content(content)
	return api.GetProfileDiagnosticsReply{Diagnostics: diag}, nil
}

func (s *SnapshotService) CreateProfileSnapshot(_ context.Context, req api.CreateProfileSnapshotRequest) (api.CreateProfileSnapshotReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
	}
	if contentID == "" {
		return api.CreateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotCreateFailed, "Profile has no draft content.", "qkboxd", true)
	}

	content, err := s.decryptContent(contentID)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, qkboxdInternalErrorMessage("Failed to decrypt content: " + err.Error())
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
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
	}
	requiredCapabilities := extractRequiredCapabilities(content)
	requiredCapabilitiesJSON, err := json.Marshal(requiredCapabilities)
	if err != nil {
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
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
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateSnapshotWithContentTx(tx, &snapshot, snapshotContent, diagJSON, nil, requiredCapabilitiesJSON)
	}); err != nil {
		return api.CreateProfileSnapshotReply{}, qkboxdInternalError(err)
	}

	return api.CreateProfileSnapshotReply{Snapshot: snapshot}, nil
}

func (s *SnapshotService) ActivateProfileSnapshot(_ context.Context, req api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if err := s.engine.CheckBlockMutations(); err != nil {
		return api.ActivateProfileSnapshotReply{}, err
	}

	snapshot, _, err := s.db.GetSnapshot(req.SnapshotID)
	if err != nil {
		return api.ActivateProfileSnapshotReply{}, qkboxdInternalError(err)
	}
	if snapshot == nil {
		return api.ActivateProfileSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveSnapshotTx(tx, snapshot.ID)
	}); err != nil {
		return api.ActivateProfileSnapshotReply{}, qkboxdInternalError(err)
	}

	return api.ActivateProfileSnapshotReply{}, nil
}

func (s *SnapshotService) GetActiveProfile(_ context.Context, _ api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError) {
	profile, err := s.db.GetActiveProfile()
	if err != nil {
		return api.GetActiveProfileReply{}, qkboxdInternalError(err)
	}
	return api.GetActiveProfileReply{Profile: profile}, nil
}

func (s *SnapshotService) GetActiveSnapshot(_ context.Context, _ api.GetActiveSnapshotRequest) (api.GetActiveSnapshotReply, *api.StructuredError) {
	snapshot, _, err := s.db.GetActiveSnapshot()
	if err != nil {
		return api.GetActiveSnapshotReply{}, qkboxdInternalError(err)
	}
	return api.GetActiveSnapshotReply{Snapshot: snapshot}, nil
}

func (s *SnapshotService) ListSnapshots(_ context.Context, req api.ListSnapshotsRequest) (api.ListSnapshotsReply, *api.StructuredError) {
	snapshots, err := s.db.ListSnapshots(req.ProfileID)
	if err != nil {
		return api.ListSnapshotsReply{}, qkboxdInternalError(err)
	}
	if snapshots == nil {
		snapshots = []model.SnapshotSummary{}
	}
	return api.ListSnapshotsReply{Snapshots: snapshots}, nil
}

func (s *SnapshotService) RollbackToSnapshot(_ context.Context, req api.RollbackToSnapshotRequest) (api.RollbackToSnapshotReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if err := s.engine.CheckBlockMutations(); err != nil {
		return api.RollbackToSnapshotReply{}, err
	}

	snapshot, _, err := s.db.GetSnapshot(req.SnapshotID)
	if err != nil {
		return api.RollbackToSnapshotReply{}, qkboxdInternalError(err)
	}
	if snapshot == nil {
		return api.RollbackToSnapshotReply{}, api.NewStructuredError(api.ErrorSnapshotNotFound, "Snapshot not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveSnapshotTx(tx, snapshot.ID)
	}); err != nil {
		return api.RollbackToSnapshotReply{}, qkboxdInternalError(err)
	}

	return api.RollbackToSnapshotReply{}, nil
}
