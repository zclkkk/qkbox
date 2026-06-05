package model

type SubscriptionUpdatePolicy string

const (
	SubscriptionUpdateManual SubscriptionUpdatePolicy = "manual"
)

type SubscriptionStatus string

const (
	SubscriptionStatusPending SubscriptionStatus = "pending"
	SubscriptionStatusUpdated SubscriptionStatus = "updated"
	SubscriptionStatusFailed  SubscriptionStatus = "failed"
)

type ProfileSubscription struct {
	ID               string                   `json:"id"`
	ProfileID        string                   `json:"profile_id"`
	Name             string                   `json:"name"`
	URL              string                   `json:"url"`
	UpdatePolicy     SubscriptionUpdatePolicy `json:"update_policy"`
	LastStatus       SubscriptionStatus       `json:"last_status"`
	LastErrorCode    string                   `json:"last_error_code,omitempty"`
	LastErrorMessage string                   `json:"last_error_message,omitempty"`
	LastCheckedAt    int64                    `json:"last_checked_at,omitempty"`
	LastUpdatedAt    int64                    `json:"last_updated_at,omitempty"`
	ContentSHA256    string                   `json:"content_sha256,omitempty"`
	CreatedAt        int64                    `json:"created_at"`
	UpdatedAt        int64                    `json:"updated_at"`
}

type DataAssetKind string

const (
	DataAssetKindRuleSet DataAssetKind = "rule_set"
	DataAssetKindGeoSite DataAssetKind = "geo_site"
	DataAssetKindGeoIP   DataAssetKind = "geo_ip"
	DataAssetKindSRSC    DataAssetKind = "srsc"
)

type DataAssetStatus string

const (
	DataAssetStatusPending   DataAssetStatus = "pending"
	DataAssetStatusAvailable DataAssetStatus = "available"
	DataAssetStatusFailed    DataAssetStatus = "failed"
)

type DataAsset struct {
	ID               string          `json:"id"`
	Kind             DataAssetKind   `json:"kind"`
	Name             string          `json:"name"`
	SourceURL        string          `json:"source_url"`
	Status           DataAssetStatus `json:"status"`
	CacheKey         string          `json:"cache_key,omitempty"`
	Version          string          `json:"version,omitempty"`
	ContentSHA256    string          `json:"content_sha256,omitempty"`
	SizeBytes        int64           `json:"size_bytes,omitempty"`
	LastErrorCode    string          `json:"last_error_code,omitempty"`
	LastErrorMessage string          `json:"last_error_message,omitempty"`
	LastCheckedAt    int64           `json:"last_checked_at,omitempty"`
	LastUpdatedAt    int64           `json:"last_updated_at,omitempty"`
	CreatedAt        int64           `json:"created_at"`
	UpdatedAt        int64           `json:"updated_at"`
}
