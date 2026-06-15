package qkboxd

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type ProfileService struct {
	db     *persistence.DB
	engine *EngineController
	opMu   *sync.Mutex
}

func (s *ProfileService) CreateProfile(_ context.Context, req api.CreateProfileRequest) (api.CreateProfileReply, *api.StructuredError) {
	if req.Name == "" {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorProfileNameEmpty, "Profile name is required.", "qkboxd", true)
	}
	if req.Content == "" {
		return api.CreateProfileReply{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is required.", "qkboxd", true)
	}
	if diag := validateProfileConfig("", req.Content); diag.Status == model.ValidationStatusInvalid {
		return api.CreateProfileReply{}, profileConfigValidationError(
			"Profile content failed validation.",
			diag,
			"Fix the profile content before creating the profile.",
		)
	}

	now := time.Now().UnixMilli()
	profile := model.Profile{
		ID:        persistence.NewProfileID(),
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateProfileTx(tx, &profile, req.Content)
	}); err != nil {
		return api.CreateProfileReply{}, qkboxdInternalError(err)
	}

	return api.CreateProfileReply{Profile: profile}, nil
}

func (s *ProfileService) UpdateProfile(_ context.Context, req api.UpdateProfileRequest) (api.UpdateProfileReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.UpdateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}
	if req.Name == "" {
		return api.UpdateProfileReply{}, api.NewStructuredError(api.ErrorProfileNameEmpty, "Profile name is required.", "qkboxd", true)
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.UpdateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateProfileTx(tx, req.ProfileID, req.Name)
	}); err != nil {
		return api.UpdateProfileReply{}, qkboxdInternalError(err)
	}

	profile, err = s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.UpdateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile disappeared after update.", "qkboxd", false)
	}
	return api.UpdateProfileReply{Profile: *profile}, nil
}

func (s *ProfileService) SaveProfileContent(_ context.Context, req api.SaveProfileContentRequest) (api.SaveProfileContentReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.SaveProfileContentReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}
	if req.Content == "" {
		return api.SaveProfileContentReply{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is required.", "qkboxd", true)
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.SaveProfileContentReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.SaveProfileContentReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}
	if diag := validateProfileConfig(req.ProfileID, req.Content); diag.Status == model.ValidationStatusInvalid {
		return api.SaveProfileContentReply{}, profileConfigValidationError(
			"Profile content failed validation.",
			diag,
			"Fix the profile content before saving it.",
		)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateProfileContentTx(tx, req.ProfileID, req.Content)
	}); err != nil {
		return api.SaveProfileContentReply{}, qkboxdInternalError(err)
	}

	profile, err = s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.SaveProfileContentReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.SaveProfileContentReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile disappeared after content save.", "qkboxd", false)
	}
	return api.SaveProfileContentReply{Profile: *profile}, nil
}

func (s *ProfileService) ValidateProfileContent(_ context.Context, req api.ValidateProfileContentRequest) (api.ValidateProfileContentReply, *api.StructuredError) {
	if req.ProfileID != "" {
		profile, err := s.db.GetProfile(req.ProfileID)
		if err != nil {
			return api.ValidateProfileContentReply{}, qkboxdInternalError(err)
		}
		if profile == nil {
			return api.ValidateProfileContentReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
		}
	}

	diag := validateProfileConfig(req.ProfileID, req.Content)
	return api.ValidateProfileContentReply{Diagnostics: diag}, nil
}

func (s *ProfileService) DeleteProfile(_ context.Context, req api.DeleteProfileRequest) (api.DeleteProfileReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.DeleteProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.db.ClearActiveProfileIfMatchesTx(tx, req.ProfileID); err != nil {
			return err
		}
		return s.db.DeleteProfileTx(tx, req.ProfileID)
	}); err != nil {
		return api.DeleteProfileReply{}, qkboxdInternalError(err)
	}

	return api.DeleteProfileReply{}, nil
}

func (s *ProfileService) ListProfiles(_ context.Context, _ api.ListProfilesRequest) (api.ListProfilesReply, *api.StructuredError) {
	profiles, err := s.db.ListProfiles()
	if err != nil {
		return api.ListProfilesReply{}, qkboxdInternalError(err)
	}
	if profiles == nil {
		profiles = []model.ProfileSummary{}
	}
	return api.ListProfilesReply{Profiles: profiles}, nil
}

func (s *ProfileService) GetProfile(_ context.Context, req api.GetProfileRequest) (api.GetProfileReply, *api.StructuredError) {
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.GetProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.GetProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	reply := api.GetProfileReply{Profile: *profile}
	content, err := s.db.GetProfileContent(req.ProfileID)
	if err != nil {
		return api.GetProfileReply{}, qkboxdInternalError(err)
	}
	reply.Content = content
	return reply, nil
}

func (s *ProfileService) ActivateProfile(_ context.Context, req api.ActivateProfileRequest) (api.ActivateProfileReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.ActivateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()

	if err := s.engine.CheckProfileSelectionMutation(); err != nil {
		return api.ActivateProfileReply{}, err
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.ActivateProfileReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.ActivateProfileReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.SetActiveProfileTx(tx, req.ProfileID)
	}); err != nil {
		return api.ActivateProfileReply{}, qkboxdInternalError(err)
	}
	return api.ActivateProfileReply{Profile: *profile}, nil
}

func (s *ProfileService) GetActiveProfile(_ context.Context, _ api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError) {
	profile, err := s.db.GetActiveProfile()
	if err != nil {
		return api.GetActiveProfileReply{}, qkboxdInternalError(err)
	}
	return api.GetActiveProfileReply{Profile: profile}, nil
}
