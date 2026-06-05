package api

import "github.com/zclkkk/qkbox/shared/model"

type CreateProfileSubscriptionRequest struct {
	ProfileID    string `json:"profile_id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	UpdatePolicy string `json:"update_policy,omitempty"`
}

type CreateProfileSubscriptionReply struct {
	Subscription model.ProfileSubscription `json:"subscription"`
}

type CreateProfileSubscriptionResult struct {
	Reply *CreateProfileSubscriptionReply `json:"reply,omitempty"`
	Error *StructuredError                `json:"error,omitempty"`
}

type ListProfileSubscriptionsRequest struct {
	ProfileID string `json:"profile_id,omitempty"`
}

type ListProfileSubscriptionsReply struct {
	Subscriptions []model.ProfileSubscription `json:"subscriptions"`
}

type ListProfileSubscriptionsResult struct {
	Reply *ListProfileSubscriptionsReply `json:"reply,omitempty"`
	Error *StructuredError               `json:"error,omitempty"`
}

type RefreshProfileSubscriptionRequest struct {
	SubscriptionID string `json:"subscription_id"`
}

type RefreshProfileSubscriptionReply struct {
	Subscription model.ProfileSubscription `json:"subscription"`
	Diagnostics  model.Diagnostics         `json:"diagnostics"`
}

type RefreshProfileSubscriptionResult struct {
	Reply *RefreshProfileSubscriptionReply `json:"reply,omitempty"`
	Error *StructuredError                 `json:"error,omitempty"`
}

type DeleteProfileSubscriptionRequest struct {
	SubscriptionID string `json:"subscription_id"`
}

type DeleteProfileSubscriptionReply struct{}

type DeleteProfileSubscriptionResult struct {
	Reply *DeleteProfileSubscriptionReply `json:"reply,omitempty"`
	Error *StructuredError                `json:"error,omitempty"`
}

type CreateDataAssetRequest struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	SourceURL string `json:"source_url"`
}

type CreateDataAssetReply struct {
	Asset model.DataAsset `json:"asset"`
}

type CreateDataAssetResult struct {
	Reply *CreateDataAssetReply `json:"reply,omitempty"`
	Error *StructuredError      `json:"error,omitempty"`
}

type ListDataAssetsRequest struct {
	Kind string `json:"kind,omitempty"`
}

type ListDataAssetsReply struct {
	Assets []model.DataAsset `json:"assets"`
}

type ListDataAssetsResult struct {
	Reply *ListDataAssetsReply `json:"reply,omitempty"`
	Error *StructuredError     `json:"error,omitempty"`
}

type RefreshDataAssetRequest struct {
	AssetID string `json:"asset_id"`
}

type RefreshDataAssetReply struct {
	Asset model.DataAsset `json:"asset"`
}

type RefreshDataAssetResult struct {
	Reply *RefreshDataAssetReply `json:"reply,omitempty"`
	Error *StructuredError       `json:"error,omitempty"`
}

type DeleteDataAssetRequest struct {
	AssetID string `json:"asset_id"`
}

type DeleteDataAssetReply struct{}

type DeleteDataAssetResult struct {
	Reply *DeleteDataAssetReply `json:"reply,omitempty"`
	Error *StructuredError      `json:"error,omitempty"`
}
