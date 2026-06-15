package qkboxd

import (
	"context"
	"database/sql"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type ProfileService struct {
	db *persistence.DB
}

func (s *ProfileService) CreateProfile(_ context.Context, req api.CreateProfileRequest) (api.CreateProfileReply, *api.StructuredError) {
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

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateProfileTx(tx, &profile, req.Content)
	}); err != nil {
		return api.CreateProfileReply{}, qkboxdInternalError(err)
	}

	return api.CreateProfileReply{Profile: profile}, nil
}

func (s *ProfileService) UpdateProfileDraft(_ context.Context, req api.UpdateProfileDraftRequest) (api.UpdateProfileDraftReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}
	if req.Content == "" {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileContentEmpty, "Profile content is required.", "qkboxd", true)
	}

	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileDraftReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateProfileContentTx(tx, req.ProfileID, req.Content)
	}); err != nil {
		return api.UpdateProfileDraftReply{}, qkboxdInternalError(err)
	}

	profile, err = s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.UpdateProfileDraftReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.UpdateProfileDraftReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile disappeared after update.", "qkboxd", false)
	}
	return api.UpdateProfileDraftReply{Profile: *profile}, nil
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
