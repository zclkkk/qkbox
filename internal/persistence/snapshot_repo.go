package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/zclkkk/qkbox/shared/model"
)

func (db *DB) InsertSnapshot(s *model.Snapshot, contentID string, diagnosticsJSON, runtimeSummaryJSON, requiredCapsJSON []byte) error {
	_, err := db.conn.Exec(
		`INSERT INTO snapshots (id, profile_id, content_id, validation_status, diagnostics_json, runtime_summary_json, required_capabilities_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ProfileID, contentID, string(s.ValidationStatus), diagnosticsJSON, runtimeSummaryJSON, requiredCapsJSON, s.CreatedAt,
	)
	return err
}

func (db *DB) GetSnapshot(id string) (*model.Snapshot, string, error) {
	var s model.Snapshot
	var contentID string
	var diagnosticsJSON, runtimeSummaryJSON, requiredCapsJSON []byte
	err := db.conn.QueryRow(
		`SELECT id, profile_id, content_id, validation_status, diagnostics_json, runtime_summary_json, required_capabilities_json, created_at
		 FROM snapshots WHERE id = ?`, id,
	).Scan(&s.ID, &s.ProfileID, &contentID, &s.ValidationStatus, &diagnosticsJSON, &runtimeSummaryJSON, &requiredCapsJSON, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if len(diagnosticsJSON) > 0 {
		_ = json.Unmarshal(diagnosticsJSON, &s.Diagnostics)
	}
	if len(runtimeSummaryJSON) > 0 {
		var rs model.RuntimeSummary
		if err := json.Unmarshal(runtimeSummaryJSON, &rs); err == nil {
			s.RuntimeSummary = &rs
		}
	}
	if len(requiredCapsJSON) > 0 {
		_ = json.Unmarshal(requiredCapsJSON, &s.RequiredCapabilities)
	}
	return &s, contentID, nil
}

func (db *DB) ListSnapshots(profileID string) ([]model.SnapshotSummary, error) {
	rows, err := db.conn.Query(
		`SELECT id, profile_id, validation_status, created_at FROM snapshots WHERE profile_id = ? ORDER BY created_at DESC`,
		profileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []model.SnapshotSummary
	for rows.Next() {
		var s model.SnapshotSummary
		if err := rows.Scan(&s.ID, &s.ProfileID, &s.ValidationStatus, &s.CreatedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

func (db *DB) GetActiveSnapshot() (*model.Snapshot, string, error) {
	var snapshotID sql.NullString
	err := db.conn.QueryRow(
		`SELECT active_snapshot_id FROM profiles WHERE active_snapshot_id IS NOT NULL LIMIT 1`,
	).Scan(&snapshotID)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	if !snapshotID.Valid {
		return nil, "", nil
	}
	return db.GetSnapshot(snapshotID.String)
}

func (db *DB) DeleteSnapshotsByProfile(profileID string) error {
	_, err := db.conn.Exec(`DELETE FROM snapshots WHERE profile_id = ?`, profileID)
	return err
}

func NewSnapshotID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("snp_%s", hex.EncodeToString(b))
}

func (db *DB) DeleteSnapshotsByProfileTx(tx *sql.Tx, profileID string) error {
	_, err := tx.Exec(`DELETE FROM snapshots WHERE profile_id = ?`, profileID)
	return err
}

func (db *DB) InsertSnapshotTx(tx *sql.Tx, s *model.Snapshot, contentID string, diagnosticsJSON, runtimeSummaryJSON, requiredCapsJSON []byte) error {
	_, err := tx.Exec(
		`INSERT INTO snapshots (id, profile_id, content_id, validation_status, diagnostics_json, runtime_summary_json, required_capabilities_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ProfileID, contentID, string(s.ValidationStatus), diagnosticsJSON, runtimeSummaryJSON, requiredCapsJSON, s.CreatedAt,
	)
	return err
}
