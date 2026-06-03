package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zclkkk/qkbox/shared/model"
)

func (db *DB) InsertProfile(p *model.Profile) error {
	_, err := db.conn.Exec(
		`INSERT INTO profiles (id, name, draft_content_id, active_snapshot_id, created_at, updated_at)
		 VALUES (?, ?, NULL, NULL, ?, ?)`,
		p.ID, p.Name, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (db *DB) GetProfile(id string) (*model.Profile, error) {
	var p model.Profile
	var draftContentID, activeSnapshotID sql.NullString
	err := db.conn.QueryRow(
		`SELECT id, name, draft_content_id, active_snapshot_id, created_at, updated_at
		 FROM profiles WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &draftContentID, &activeSnapshotID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if activeSnapshotID.Valid {
		p.ActiveSnapshotID = &activeSnapshotID.String
	}
	return &p, nil
}

func (db *DB) UpdateProfileDraftContent(profileID, contentID string) error {
	now := time.Now().UnixMilli()
	_, err := db.conn.Exec(
		`UPDATE profiles SET draft_content_id = ?, updated_at = ? WHERE id = ?`,
		contentID, now, profileID,
	)
	return err
}

func (db *DB) UpdateProfileActiveSnapshot(profileID string, snapshotID *string) error {
	now := time.Now().UnixMilli()
	var id sql.NullString
	if snapshotID != nil {
		id = sql.NullString{String: *snapshotID, Valid: true}
	}
	_, err := db.conn.Exec(
		`UPDATE profiles SET active_snapshot_id = ?, updated_at = ? WHERE id = ?`,
		id, now, profileID,
	)
	return err
}

func (db *DB) ListProfiles() ([]model.ProfileSummary, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, draft_content_id, active_snapshot_id, created_at, updated_at FROM profiles ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []model.ProfileSummary
	for rows.Next() {
		var p model.ProfileSummary
		var draftContentID, activeSnapshotID sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &draftContentID, &activeSnapshotID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.HasDraft = draftContentID.Valid
		p.HasActiveSnapshot = activeSnapshotID.Valid
		if activeSnapshotID.Valid {
			p.ActiveSnapshotID = &activeSnapshotID.String
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (db *DB) DeleteProfile(id string) error {
	_, err := db.conn.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	return err
}

func (db *DB) GetProfileDraftContentID(profileID string) (string, error) {
	var contentID sql.NullString
	err := db.conn.QueryRow(
		`SELECT draft_content_id FROM profiles WHERE id = ?`, profileID,
	).Scan(&contentID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !contentID.Valid {
		return "", nil
	}
	return contentID.String, nil
}

func (db *DB) GetProfileDraftContentIDTx(tx *sql.Tx, profileID string) (string, error) {
	var contentID sql.NullString
	err := tx.QueryRow(
		`SELECT draft_content_id FROM profiles WHERE id = ?`, profileID,
	).Scan(&contentID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !contentID.Valid {
		return "", nil
	}
	return contentID.String, nil
}

func (db *DB) GetActiveProfile() (*model.Profile, error) {
	var p model.Profile
	var draftContentID, activeSnapshotID sql.NullString
	err := db.conn.QueryRow(
		`SELECT id, name, draft_content_id, active_snapshot_id, created_at, updated_at
		 FROM profiles WHERE active_snapshot_id IS NOT NULL LIMIT 1`,
	).Scan(&p.ID, &p.Name, &draftContentID, &activeSnapshotID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if activeSnapshotID.Valid {
		p.ActiveSnapshotID = &activeSnapshotID.String
	}
	return &p, nil
}

func NewProfileID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("prf_%s", hex.EncodeToString(b))
}

func (db *DB) InsertProfileTx(tx *sql.Tx, p *model.Profile) error {
	_, err := tx.Exec(
		`INSERT INTO profiles (id, name, draft_content_id, active_snapshot_id, created_at, updated_at)
		 VALUES (?, ?, NULL, NULL, ?, ?)`,
		p.ID, p.Name, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (db *DB) UpdateProfileDraftContentTx(tx *sql.Tx, profileID, contentID string) error {
	now := time.Now().UnixMilli()
	_, err := tx.Exec(
		`UPDATE profiles SET draft_content_id = ?, updated_at = ? WHERE id = ?`,
		contentID, now, profileID,
	)
	return err
}

func (db *DB) UpdateProfileActiveSnapshotTx(tx *sql.Tx, profileID string, snapshotID *string) error {
	now := time.Now().UnixMilli()
	var id sql.NullString
	if snapshotID != nil {
		id = sql.NullString{String: *snapshotID, Valid: true}
	}
	_, err := tx.Exec(
		`UPDATE profiles SET active_snapshot_id = ?, updated_at = ? WHERE id = ?`,
		id, now, profileID,
	)
	return err
}

func (db *DB) ClearAllActiveSnapshotsTx(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE profiles SET active_snapshot_id = NULL WHERE active_snapshot_id IS NOT NULL`)
	return err
}

func (db *DB) DeleteProfileTx(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	return err
}
