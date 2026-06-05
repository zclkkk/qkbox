package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zclkkk/qkbox/shared/model"
)

type ProfileSubscriptionUpdate struct {
	LastStatus       model.SubscriptionStatus
	LastErrorCode    string
	LastErrorMessage string
	LastCheckedAt    int64
	LastUpdatedAt    int64
	ContentSHA256    string
}

type DataAssetUpdate struct {
	Status           model.DataAssetStatus
	CacheKey         string
	Version          string
	ContentSHA256    string
	SizeBytes        int64
	LastErrorCode    string
	LastErrorMessage string
	LastCheckedAt    int64
	LastUpdatedAt    int64
}

func NewProfileSubscriptionID() string {
	return prefixedRandomID("sub")
}

func NewDataAssetID() string {
	return prefixedRandomID("ast")
}

func prefixedRandomID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func (db *DB) CreateProfileSubscriptionTx(tx *sql.Tx, sub *model.ProfileSubscription) error {
	_, err := tx.Exec(
		`INSERT INTO profile_subscriptions (
			id, profile_id, name, url, update_policy, last_status,
			last_error_code, last_error_message, last_checked_at, last_updated_at,
			content_sha256, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID, sub.ProfileID, sub.Name, sub.URL, string(sub.UpdatePolicy), string(sub.LastStatus),
		nullableString(sub.LastErrorCode), nullableString(sub.LastErrorMessage), nullableInt64(sub.LastCheckedAt), nullableInt64(sub.LastUpdatedAt),
		nullableString(sub.ContentSHA256), sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

func (db *DB) GetProfileSubscription(id string) (*model.ProfileSubscription, error) {
	row := db.conn.QueryRow(
		`SELECT id, profile_id, name, url, update_policy, last_status,
		        last_error_code, last_error_message, last_checked_at, last_updated_at,
		        content_sha256, created_at, updated_at
		   FROM profile_subscriptions WHERE id = ?`,
		id,
	)
	return scanProfileSubscription(row)
}

func (db *DB) ListProfileSubscriptions(profileID string) ([]model.ProfileSubscription, error) {
	query := `SELECT id, profile_id, name, url, update_policy, last_status,
	                 last_error_code, last_error_message, last_checked_at, last_updated_at,
	                 content_sha256, created_at, updated_at
	            FROM profile_subscriptions`
	args := []interface{}{}
	if profileID != "" {
		query += ` WHERE profile_id = ?`
		args = append(args, profileID)
	}
	query += ` ORDER BY created_at`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.ProfileSubscription
	for rows.Next() {
		sub, err := scanProfileSubscriptionRows(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *sub)
	}
	return subs, rows.Err()
}

func (db *DB) UpdateProfileSubscriptionRefreshTx(tx *sql.Tx, id string, update ProfileSubscriptionUpdate) error {
	now := time.Now().UnixMilli()
	_, err := tx.Exec(
		`UPDATE profile_subscriptions
		    SET last_status = ?,
		        last_error_code = ?,
		        last_error_message = ?,
		        last_checked_at = ?,
		        last_updated_at = ?,
		        content_sha256 = ?,
		        updated_at = ?
		  WHERE id = ?`,
		string(update.LastStatus),
		nullableString(update.LastErrorCode),
		nullableString(update.LastErrorMessage),
		nullableInt64(update.LastCheckedAt),
		nullableInt64(update.LastUpdatedAt),
		nullableString(update.ContentSHA256),
		now,
		id,
	)
	return err
}

func (db *DB) DeleteProfileSubscriptionTx(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM profile_subscriptions WHERE id = ?`, id)
	return err
}

func (db *DB) CreateDataAssetTx(tx *sql.Tx, asset *model.DataAsset) error {
	_, err := tx.Exec(
		`INSERT INTO data_assets (
			id, kind, name, source_url, status, cache_key, version,
			content_sha256, size_bytes, last_error_code, last_error_message,
			last_checked_at, last_updated_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		asset.ID, string(asset.Kind), asset.Name, asset.SourceURL, string(asset.Status),
		nullableString(asset.CacheKey), nullableString(asset.Version), nullableString(asset.ContentSHA256),
		nullableInt64(asset.SizeBytes), nullableString(asset.LastErrorCode), nullableString(asset.LastErrorMessage),
		nullableInt64(asset.LastCheckedAt), nullableInt64(asset.LastUpdatedAt), asset.CreatedAt, asset.UpdatedAt,
	)
	return err
}

func (db *DB) GetDataAsset(id string) (*model.DataAsset, error) {
	row := db.conn.QueryRow(
		`SELECT id, kind, name, source_url, status, cache_key, version,
		        content_sha256, size_bytes, last_error_code, last_error_message,
		        last_checked_at, last_updated_at, created_at, updated_at
		   FROM data_assets WHERE id = ?`,
		id,
	)
	return scanDataAsset(row)
}

func (db *DB) ListDataAssets(kind string) ([]model.DataAsset, error) {
	query := `SELECT id, kind, name, source_url, status, cache_key, version,
	                 content_sha256, size_bytes, last_error_code, last_error_message,
	                 last_checked_at, last_updated_at, created_at, updated_at
	            FROM data_assets`
	args := []interface{}{}
	if kind != "" {
		query += ` WHERE kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY created_at`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []model.DataAsset
	for rows.Next() {
		asset, err := scanDataAssetRows(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *asset)
	}
	return assets, rows.Err()
}

func (db *DB) UpdateDataAssetRefreshTx(tx *sql.Tx, id string, update DataAssetUpdate) error {
	now := time.Now().UnixMilli()
	_, err := tx.Exec(
		`UPDATE data_assets
		    SET status = ?,
		        cache_key = ?,
		        version = ?,
		        content_sha256 = ?,
		        size_bytes = ?,
		        last_error_code = ?,
		        last_error_message = ?,
		        last_checked_at = ?,
		        last_updated_at = ?,
		        updated_at = ?
		  WHERE id = ?`,
		string(update.Status),
		nullableString(update.CacheKey),
		nullableString(update.Version),
		nullableString(update.ContentSHA256),
		nullableInt64(update.SizeBytes),
		nullableString(update.LastErrorCode),
		nullableString(update.LastErrorMessage),
		nullableInt64(update.LastCheckedAt),
		nullableInt64(update.LastUpdatedAt),
		now,
		id,
	)
	return err
}

func (db *DB) DeleteDataAssetTx(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM data_assets WHERE id = ?`, id)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanProfileSubscription(row scanner) (*model.ProfileSubscription, error) {
	var sub model.ProfileSubscription
	var updatePolicy, status string
	var lastErrorCode, lastErrorMessage, lastContentSHA256 sql.NullString
	var lastCheckedAt, lastUpdatedAt sql.NullInt64
	err := row.Scan(
		&sub.ID, &sub.ProfileID, &sub.Name, &sub.URL, &updatePolicy, &status,
		&lastErrorCode, &lastErrorMessage, &lastCheckedAt, &lastUpdatedAt,
		&lastContentSHA256, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sub.UpdatePolicy = model.SubscriptionUpdatePolicy(updatePolicy)
	sub.LastStatus = model.SubscriptionStatus(status)
	sub.LastErrorCode = nullString(lastErrorCode)
	sub.LastErrorMessage = nullString(lastErrorMessage)
	sub.LastCheckedAt = nullInt64(lastCheckedAt)
	sub.LastUpdatedAt = nullInt64(lastUpdatedAt)
	sub.ContentSHA256 = nullString(lastContentSHA256)
	return &sub, nil
}

func scanProfileSubscriptionRows(rows *sql.Rows) (*model.ProfileSubscription, error) {
	return scanProfileSubscription(rows)
}

func scanDataAsset(row scanner) (*model.DataAsset, error) {
	var asset model.DataAsset
	var kind, status string
	var cacheKey, version, contentSHA, lastErrorCode, lastErrorMessage sql.NullString
	var sizeBytes, lastCheckedAt, lastUpdatedAt sql.NullInt64
	err := row.Scan(
		&asset.ID, &kind, &asset.Name, &asset.SourceURL, &status, &cacheKey, &version,
		&contentSHA, &sizeBytes, &lastErrorCode, &lastErrorMessage,
		&lastCheckedAt, &lastUpdatedAt, &asset.CreatedAt, &asset.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	asset.Kind = model.DataAssetKind(kind)
	asset.Status = model.DataAssetStatus(status)
	asset.CacheKey = nullString(cacheKey)
	asset.Version = nullString(version)
	asset.ContentSHA256 = nullString(contentSHA)
	asset.SizeBytes = nullInt64(sizeBytes)
	asset.LastErrorCode = nullString(lastErrorCode)
	asset.LastErrorMessage = nullString(lastErrorMessage)
	asset.LastCheckedAt = nullInt64(lastCheckedAt)
	asset.LastUpdatedAt = nullInt64(lastUpdatedAt)
	return &asset, nil
}

func scanDataAssetRows(rows *sql.Rows) (*model.DataAsset, error) {
	return scanDataAsset(rows)
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt64(value int64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullInt64(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}
