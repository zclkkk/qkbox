package qkboxd

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/zclkkk/qkbox/internal/assetcache"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/redact"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AssetService struct {
	*ContentCodec
	db         *persistence.DB
	httpClient *http.Client
	assetStore *assetcache.Store
}

func (s *AssetService) CreateProfileSubscription(_ context.Context, req api.CreateProfileSubscriptionRequest) (api.CreateProfileSubscriptionReply, *api.StructuredError) {
	if req.ProfileID == "" {
		return api.CreateProfileSubscriptionReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile ID is required.", "qkboxd", true)
	}
	profile, err := s.db.GetProfile(req.ProfileID)
	if err != nil {
		return api.CreateProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.CreateProfileSubscriptionReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Profile not found.", "qkboxd", true)
	}
	remoteURL, structured := validateRemoteURL(req.URL, api.ErrorSubscriptionURLEmpty, api.ErrorSubscriptionURLInvalid)
	if structured != nil {
		return api.CreateProfileSubscriptionReply{}, structured
	}
	policy, structured := normalizeSubscriptionUpdatePolicy(req.UpdatePolicy)
	if structured != nil {
		return api.CreateProfileSubscriptionReply{}, structured
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = remoteURL.Host
	}

	now := time.Now().UnixMilli()
	sub := model.ProfileSubscription{
		ID:           persistence.NewProfileSubscriptionID(),
		ProfileID:    req.ProfileID,
		Name:         name,
		URL:          remoteURL.String(),
		UpdatePolicy: policy,
		LastStatus:   model.SubscriptionStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateProfileSubscriptionTx(tx, &sub)
	}); err != nil {
		return api.CreateProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	return api.CreateProfileSubscriptionReply{Subscription: sub}, nil
}

func (s *AssetService) ListProfileSubscriptions(_ context.Context, req api.ListProfileSubscriptionsRequest) (api.ListProfileSubscriptionsReply, *api.StructuredError) {
	subs, err := s.db.ListProfileSubscriptions(req.ProfileID)
	if err != nil {
		return api.ListProfileSubscriptionsReply{}, qkboxdInternalError(err)
	}
	if subs == nil {
		subs = []model.ProfileSubscription{}
	}
	return api.ListProfileSubscriptionsReply{Subscriptions: subs}, nil
}

func (s *AssetService) RefreshProfileSubscription(ctx context.Context, req api.RefreshProfileSubscriptionRequest) (api.RefreshProfileSubscriptionReply, *api.StructuredError) {
	sub, err := s.db.GetProfileSubscription(req.SubscriptionID)
	if err != nil {
		return api.RefreshProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	if sub == nil {
		return api.RefreshProfileSubscriptionReply{}, api.NewStructuredError(api.ErrorSubscriptionNotFound, "Subscription not found.", "qkboxd", true)
	}
	profile, err := s.db.GetProfile(sub.ProfileID)
	if err != nil {
		return api.RefreshProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	if profile == nil {
		return api.RefreshProfileSubscriptionReply{}, api.NewStructuredError(api.ErrorProfileNotFound, "Subscription profile not found.", "qkboxd", true)
	}

	fetched, structured := s.fetchRemote(ctx, sub.URL, remoteProfileFetchLimit, api.ErrorSubscriptionFetchFailed)
	checkedAt := time.Now().UnixMilli()
	if structured != nil {
		_ = s.recordProfileSubscriptionFailure(req.SubscriptionID, checkedAt, structured.Code, structured.Message)
		return api.RefreshProfileSubscriptionReply{}, structured
	}

	content := string(fetched.Content)
	diag := validateContent(content)
	diag.ProfileID = sub.ProfileID
	diag.RedactedPreview = redact.Content(content)
	if diag.Status == model.ValidationStatusInvalid {
		structured := &api.StructuredError{
			Code:        api.ErrorConfigValidationFailed,
			Message:     "Subscription content failed profile validation.",
			Detail:      diag.Entries,
			Source:      "qkboxd",
			Recoverable: true,
			UserAction:  "Fix the remote subscription content before refreshing again.",
		}
		_ = s.recordProfileSubscriptionFailure(req.SubscriptionID, checkedAt, structured.Code, structured.Message)
		return api.RefreshProfileSubscriptionReply{}, structured
	}

	contentSHA := sha256Hex(fetched.Content)
	draftContent, err := s.encryptedContent("draft", sub.ProfileID, content, checkedAt)
	if err != nil {
		return api.RefreshProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	update := persistence.ProfileSubscriptionUpdate{
		LastStatus:    model.SubscriptionStatusUpdated,
		LastCheckedAt: checkedAt,
		LastUpdatedAt: checkedAt,
		ContentSHA256: contentSHA,
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		if err := s.db.ReplaceDraftContentTx(tx, sub.ProfileID, draftContent); err != nil {
			return err
		}
		return s.db.UpdateProfileSubscriptionRefreshTx(tx, req.SubscriptionID, update)
	}); err != nil {
		return api.RefreshProfileSubscriptionReply{}, qkboxdInternalError(err)
	}

	refreshed, err := s.db.GetProfileSubscription(req.SubscriptionID)
	if err != nil {
		return api.RefreshProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	return api.RefreshProfileSubscriptionReply{Subscription: *refreshed, Diagnostics: diag}, nil
}

func (s *AssetService) DeleteProfileSubscription(_ context.Context, req api.DeleteProfileSubscriptionRequest) (api.DeleteProfileSubscriptionReply, *api.StructuredError) {
	sub, err := s.db.GetProfileSubscription(req.SubscriptionID)
	if err != nil {
		return api.DeleteProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	if sub == nil {
		return api.DeleteProfileSubscriptionReply{}, api.NewStructuredError(api.ErrorSubscriptionNotFound, "Subscription not found.", "qkboxd", true)
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.DeleteProfileSubscriptionTx(tx, req.SubscriptionID)
	}); err != nil {
		return api.DeleteProfileSubscriptionReply{}, qkboxdInternalError(err)
	}
	return api.DeleteProfileSubscriptionReply{}, nil
}

func (s *AssetService) CreateDataAsset(_ context.Context, req api.CreateDataAssetRequest) (api.CreateDataAssetReply, *api.StructuredError) {
	kind, structured := normalizeDataAssetKind(req.Kind)
	if structured != nil {
		return api.CreateDataAssetReply{}, structured
	}
	remoteURL, structured := validateRemoteURL(req.SourceURL, api.ErrorAssetSourceURLInvalid, api.ErrorAssetSourceURLInvalid)
	if structured != nil {
		return api.CreateDataAssetReply{}, structured
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = remoteURL.Host
	}
	now := time.Now().UnixMilli()
	asset := model.DataAsset{
		ID:        persistence.NewDataAssetID(),
		Kind:      kind,
		Name:      name,
		SourceURL: remoteURL.String(),
		Status:    model.DataAssetStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.CreateDataAssetTx(tx, &asset)
	}); err != nil {
		return api.CreateDataAssetReply{}, qkboxdInternalError(err)
	}
	return api.CreateDataAssetReply{Asset: asset}, nil
}

func (s *AssetService) ListDataAssets(_ context.Context, req api.ListDataAssetsRequest) (api.ListDataAssetsReply, *api.StructuredError) {
	if req.Kind != "" {
		if _, structured := normalizeDataAssetKind(req.Kind); structured != nil {
			return api.ListDataAssetsReply{}, structured
		}
	}
	assets, err := s.db.ListDataAssets(req.Kind)
	if err != nil {
		return api.ListDataAssetsReply{}, qkboxdInternalError(err)
	}
	if assets == nil {
		assets = []model.DataAsset{}
	}
	return api.ListDataAssetsReply{Assets: assets}, nil
}

func (s *AssetService) RefreshDataAsset(ctx context.Context, req api.RefreshDataAssetRequest) (api.RefreshDataAssetReply, *api.StructuredError) {
	asset, err := s.db.GetDataAsset(req.AssetID)
	if err != nil {
		return api.RefreshDataAssetReply{}, qkboxdInternalError(err)
	}
	if asset == nil {
		return api.RefreshDataAssetReply{}, api.NewStructuredError(api.ErrorAssetNotFound, "Data asset not found.", "qkboxd", true)
	}

	fetched, structured := s.fetchRemote(ctx, asset.SourceURL, dataAssetFetchLimit, api.ErrorAssetFetchFailed)
	checkedAt := time.Now().UnixMilli()
	if structured != nil {
		_ = s.recordDataAssetFailure(req.AssetID, checkedAt, structured.Code, structured.Message)
		return api.RefreshDataAssetReply{}, structured
	}
	if structured := validateDataAssetContent(asset.Kind, fetched.Content); structured != nil {
		_ = s.recordDataAssetFailure(req.AssetID, checkedAt, structured.Code, structured.Message)
		return api.RefreshDataAssetReply{}, structured
	}

	blob, err := s.assetStore.Put(string(asset.Kind), fetched.Content)
	if err != nil {
		structured := api.NewStructuredError(api.ErrorAssetCacheFailed, err.Error(), "qkboxd", true)
		_ = s.recordDataAssetFailure(req.AssetID, checkedAt, structured.Code, structured.Message)
		return api.RefreshDataAssetReply{}, structured
	}
	update := persistence.DataAssetUpdate{
		Status:        model.DataAssetStatusAvailable,
		CacheKey:      blob.Key,
		Version:       fetched.Version,
		ContentSHA256: blob.SHA256,
		SizeBytes:     blob.SizeBytes,
		LastCheckedAt: checkedAt,
		LastUpdatedAt: checkedAt,
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateDataAssetRefreshTx(tx, req.AssetID, update)
	}); err != nil {
		return api.RefreshDataAssetReply{}, qkboxdInternalError(err)
	}
	refreshed, err := s.db.GetDataAsset(req.AssetID)
	if err != nil {
		return api.RefreshDataAssetReply{}, qkboxdInternalError(err)
	}
	return api.RefreshDataAssetReply{Asset: *refreshed}, nil
}

func (s *AssetService) DeleteDataAsset(_ context.Context, req api.DeleteDataAssetRequest) (api.DeleteDataAssetReply, *api.StructuredError) {
	asset, err := s.db.GetDataAsset(req.AssetID)
	if err != nil {
		return api.DeleteDataAssetReply{}, qkboxdInternalError(err)
	}
	if asset == nil {
		return api.DeleteDataAssetReply{}, api.NewStructuredError(api.ErrorAssetNotFound, "Data asset not found.", "qkboxd", true)
	}
	if err := s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.DeleteDataAssetTx(tx, req.AssetID)
	}); err != nil {
		return api.DeleteDataAssetReply{}, qkboxdInternalError(err)
	}
	return api.DeleteDataAssetReply{}, nil
}

type fetchedRemoteContent struct {
	Content []byte
	Version string
}

func (s *AssetService) fetchRemote(ctx context.Context, rawURL string, limit int64, errorCode string) (fetchedRemoteContent, *api.StructuredError) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, err.Error(), "qkboxd", true)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, err.Error(), "qkboxd", true)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, fmt.Sprintf("remote returned HTTP %d", resp.StatusCode), "qkboxd", true)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, err.Error(), "qkboxd", true)
	}
	if int64(len(body)) > limit {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, "remote content exceeds qkbox size limit", "qkboxd", true)
	}
	if len(body) == 0 {
		return fetchedRemoteContent{}, api.NewStructuredError(errorCode, "remote content is empty", "qkboxd", true)
	}
	return fetchedRemoteContent{Content: body, Version: remoteVersion(resp)}, nil
}

func remoteVersion(resp *http.Response) string {
	if etag := strings.TrimSpace(resp.Header.Get("ETag")); etag != "" {
		return etag
	}
	return strings.TrimSpace(resp.Header.Get("Last-Modified"))
}

func validateRemoteURL(value string, emptyCode string, invalidCode string) (*url.URL, *api.StructuredError) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, api.NewStructuredError(emptyCode, "URL is required.", "qkboxd", true)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return nil, api.NewStructuredError(invalidCode, "URL must be an absolute HTTP or HTTPS URL.", "qkboxd", true)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, api.NewStructuredError(invalidCode, "URL must use HTTP or HTTPS.", "qkboxd", true)
	}
	return parsed, nil
}

func normalizeSubscriptionUpdatePolicy(value string) (model.SubscriptionUpdatePolicy, *api.StructuredError) {
	if value == "" {
		return model.SubscriptionUpdateManual, nil
	}
	policy := model.SubscriptionUpdatePolicy(value)
	if policy != model.SubscriptionUpdateManual {
		return "", api.NewStructuredError(api.ErrorIPCInvalidRequest, "Unsupported subscription update policy.", "qkboxd", true)
	}
	return policy, nil
}

func normalizeDataAssetKind(value string) (model.DataAssetKind, *api.StructuredError) {
	kind := model.DataAssetKind(strings.TrimSpace(value))
	switch kind {
	case model.DataAssetKindRuleSet, model.DataAssetKindGeoSite, model.DataAssetKindGeoIP, model.DataAssetKindSRSC:
		return kind, nil
	default:
		return "", api.NewStructuredError(api.ErrorAssetKindUnsupported, "Unsupported data asset kind.", "qkboxd", true)
	}
}

func validateDataAssetContent(kind model.DataAssetKind, content []byte) *api.StructuredError {
	if len(content) == 0 {
		return api.NewStructuredError(api.ErrorAssetValidationFailed, "Data asset content is empty.", "qkboxd", true)
	}
	if kind == model.DataAssetKindRuleSet {
		var raw interface{}
		if err := json.Unmarshal(content, &raw); err != nil {
			return api.NewStructuredError(api.ErrorAssetValidationFailed, "Rule-set asset is not valid JSON: "+err.Error(), "qkboxd", true)
		}
		switch raw.(type) {
		case map[string]interface{}, []interface{}:
			return nil
		default:
			return api.NewStructuredError(api.ErrorAssetValidationFailed, "Rule-set asset must be a JSON object or array.", "qkboxd", true)
		}
	}
	return nil
}

func (s *AssetService) recordProfileSubscriptionFailure(id string, checkedAt int64, code string, message string) error {
	return s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateProfileSubscriptionRefreshTx(tx, id, persistence.ProfileSubscriptionUpdate{
			LastStatus:       model.SubscriptionStatusFailed,
			LastErrorCode:    code,
			LastErrorMessage: message,
			LastCheckedAt:    checkedAt,
		})
	})
}

func (s *AssetService) recordDataAssetFailure(id string, checkedAt int64, code string, message string) error {
	return s.db.WithTx(func(tx *sql.Tx) error {
		return s.db.UpdateDataAssetRefreshTx(tx, id, persistence.DataAssetUpdate{
			Status:           model.DataAssetStatusFailed,
			LastErrorCode:    code,
			LastErrorMessage: message,
			LastCheckedAt:    checkedAt,
		})
	})
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
