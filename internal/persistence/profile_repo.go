package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zclkkk/qkbox/shared/model"
)

const activeProfileSettingKey = "active_profile_id"

func (db *DB) GetProfile(id string) (*model.Profile, error) {
	var p model.Profile
	err := db.conn.QueryRow(
		`SELECT id, name, created_at, updated_at
		 FROM profiles
		 WHERE id = ?`,
		id,
	).Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) ListProfiles() ([]model.ProfileSummary, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, created_at, updated_at
		 FROM profiles
		 ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []model.ProfileSummary
	for rows.Next() {
		var p model.ProfileSummary
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (db *DB) GetProfileContent(profileID string) (string, error) {
	var content string
	err := db.conn.QueryRow(
		`SELECT content FROM profiles WHERE id = ?`,
		profileID,
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return content, nil
}

func (db *DB) GetActiveProfileID() (string, error) {
	value, err := db.GetSetting(activeProfileSettingKey)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func (db *DB) GetActiveProfile() (*model.Profile, error) {
	profileID, err := db.GetActiveProfileID()
	if err != nil {
		return nil, err
	}
	if profileID == "" {
		return nil, nil
	}
	return db.GetProfile(profileID)
}

func NewProfileID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("prf_%s", hex.EncodeToString(b))
}

func (db *DB) CreateProfileTx(tx *sql.Tx, p *model.Profile, content string) error {
	_, err := tx.Exec(
		`INSERT INTO profiles (id, name, content, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Name, content, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (db *DB) UpdateProfileContentTx(tx *sql.Tx, profileID, content string) error {
	now := time.Now().UnixMilli()
	_, err := tx.Exec(
		`UPDATE profiles SET content = ?, updated_at = ? WHERE id = ?`,
		content, now, profileID,
	)
	return err
}

func (db *DB) SetActiveProfileTx(tx *sql.Tx, profileID string) error {
	_, err := tx.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		activeProfileSettingKey, profileID,
	)
	return err
}

func (db *DB) ClearActiveProfileTx(tx *sql.Tx) error {
	_, err := tx.Exec(`DELETE FROM settings WHERE key = ?`, activeProfileSettingKey)
	return err
}

func (db *DB) ClearActiveProfileIfMatchesTx(tx *sql.Tx, profileID string) error {
	_, err := tx.Exec(`DELETE FROM settings WHERE key = ? AND value = ?`, activeProfileSettingKey, profileID)
	return err
}

func (db *DB) DeleteProfileTx(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	return err
}
