package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zclkkk/qkbox/shared/model"
)

func (db *DB) GetProfile(id string) (*model.Profile, error) {
	var p model.Profile
	var activeSnapshotID sql.NullString
	err := db.conn.QueryRow(
		`SELECT p.id, p.name,
		        CASE WHEN s.profile_id = p.id THEN rs.active_snapshot_id ELSE NULL END,
		        p.created_at, p.updated_at
		 FROM profiles p
		 LEFT JOIN runtime_state rs ON rs.id = 1
		 LEFT JOIN snapshots s ON s.id = rs.active_snapshot_id
		 WHERE p.id = ?`,
		id,
	).Scan(&p.ID, &p.Name, &activeSnapshotID, &p.CreatedAt, &p.UpdatedAt)
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

func (db *DB) ListProfiles() ([]model.ProfileSummary, error) {
	rows, err := db.conn.Query(
		`SELECT p.id, p.name, p.draft_content_id,
		        CASE WHEN s.profile_id = p.id THEN rs.active_snapshot_id ELSE NULL END,
		        p.created_at, p.updated_at
		 FROM profiles p
		 LEFT JOIN runtime_state rs ON rs.id = 1
		 LEFT JOIN snapshots s ON s.id = rs.active_snapshot_id
		 ORDER BY p.created_at`,
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

func (db *DB) GetActiveProfile() (*model.Profile, error) {
	var p model.Profile
	var activeSnapshotID sql.NullString
	err := db.conn.QueryRow(
		`SELECT p.id, p.name, rs.active_snapshot_id, p.created_at, p.updated_at
		 FROM runtime_state rs
		 JOIN snapshots s ON s.id = rs.active_snapshot_id
		 JOIN profiles p ON p.id = s.profile_id
		 WHERE rs.id = 1 AND rs.active_snapshot_id IS NOT NULL`,
	).Scan(&p.ID, &p.Name, &activeSnapshotID, &p.CreatedAt, &p.UpdatedAt)
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

func (db *DB) CreateProfileWithDraftTx(tx *sql.Tx, p *model.Profile, content *EncryptedContent) error {
	if err := db.insertProfileTx(tx, p); err != nil {
		return err
	}
	if err := db.insertContentTx(tx, content); err != nil {
		return err
	}
	return db.updateProfileDraftContentTx(tx, p.ID, content.ID)
}

func (db *DB) ReplaceDraftContentTx(tx *sql.Tx, profileID string, content *EncryptedContent) error {
	if err := db.deleteContentBySourceTx(tx, "draft", profileID); err != nil {
		return err
	}
	if err := db.insertContentTx(tx, content); err != nil {
		return err
	}
	return db.updateProfileDraftContentTx(tx, profileID, content.ID)
}

func (db *DB) DeleteProfileGraphTx(tx *sql.Tx, profileID string) error {
	if err := db.deleteSnapshotsByProfileTx(tx, profileID); err != nil {
		return err
	}
	if err := db.deleteContentBySourceTx(tx, "draft", profileID); err != nil {
		return err
	}
	if err := db.deleteContentBySourceTx(tx, "snapshot", profileID); err != nil {
		return err
	}
	return db.deleteProfileTx(tx, profileID)
}

func (db *DB) insertProfileTx(tx *sql.Tx, p *model.Profile) error {
	_, err := tx.Exec(
		`INSERT INTO profiles (id, name, draft_content_id, created_at, updated_at)
		 VALUES (?, ?, NULL, ?, ?)`,
		p.ID, p.Name, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (db *DB) updateProfileDraftContentTx(tx *sql.Tx, profileID, contentID string) error {
	now := time.Now().UnixMilli()
	_, err := tx.Exec(
		`UPDATE profiles SET draft_content_id = ?, updated_at = ? WHERE id = ?`,
		contentID, now, profileID,
	)
	return err
}

func (db *DB) deleteProfileTx(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	return err
}
