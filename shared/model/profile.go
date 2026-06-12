package model

type Profile struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	ActiveSnapshotID *string `json:"active_snapshot_id,omitempty"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

type ProfileSummary struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	HasDraft          bool    `json:"has_draft"`
	HasActiveSnapshot bool    `json:"has_active_snapshot"`
	ActiveSnapshotID  *string `json:"active_snapshot_id,omitempty"`
	CreatedAt         int64   `json:"created_at"`
	UpdatedAt         int64   `json:"updated_at"`
}
