package persistence

import (
	"database/sql"
	"time"
)

func (db *DB) GetActiveSnapshotID() (string, error) {
	var snapshotID sql.NullString
	err := db.conn.QueryRow(`SELECT active_snapshot_id FROM runtime_state WHERE id = 1`).Scan(&snapshotID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !snapshotID.Valid {
		return "", nil
	}
	return snapshotID.String, nil
}

func (db *DB) SetActiveSnapshotTx(tx *sql.Tx, snapshotID string) error {
	_, err := tx.Exec(
		`INSERT INTO runtime_state (id, active_snapshot_id, updated_at)
		 VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   active_snapshot_id = excluded.active_snapshot_id,
		   updated_at = excluded.updated_at`,
		snapshotID, time.Now().UnixMilli(),
	)
	return err
}
