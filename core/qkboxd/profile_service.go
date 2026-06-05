package qkboxd

import (
	"context"
	"database/sql"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
	"time"
)

type ProfileService struct {
	*ContentCodec
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

	draftContent, err := s.encryptedContent("draft", profile.ID, req.Content, now)
	if err != nil {
		return api.CreateProfileReply{}, qkboxdInternalError(err)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateProfileWithDraftTx(tx, &profile, draftContent)
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

	draftContent, err := s.encryptedContent("draft", req.ProfileID, req.Content, time.Now().UnixMilli())
	if err != nil {
		return api.UpdateProfileDraftReply{}, qkboxdInternalError(err)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.ReplaceDraftContentTx(tx, req.ProfileID, draftContent)
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
	if profile.ActiveSnapshotID != nil {
		return api.DeleteProfileReply{}, api.NewStructuredError(api.ErrorProfileHasSnapshot, "Deactivate the active snapshot before deleting.", "qkboxd", true)
	}

	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.DeleteProfileGraphTx(tx, req.ProfileID)
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
	contentID, err := s.db.GetProfileDraftContentID(req.ProfileID)
	if err != nil {
		return api.GetProfileReply{}, qkboxdInternalError(err)
	}
	if contentID != "" {
		content, err := s.decryptContent(contentID)
		if err != nil {
			return api.GetProfileReply{}, qkboxdInternalErrorMessage("Failed to decrypt draft content: " + err.Error())
		}
		reply.Content = content
	}
	return reply, nil
}
