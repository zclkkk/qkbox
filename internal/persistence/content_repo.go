package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type EncryptedContent struct {
	ID         string
	SourceType string
	SourceID   string
	IV         []byte
	Ciphertext []byte
	CreatedAt  int64
}

func (db *DB) insertContentTx(tx *sql.Tx, c *EncryptedContent) error {
	_, err := tx.Exec(
		`INSERT INTO encrypted_content (id, source_type, source_id, iv, ciphertext, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.SourceType, c.SourceID, c.IV, c.Ciphertext, c.CreatedAt,
	)
	return err
}

func (db *DB) GetContent(id string) (*EncryptedContent, error) {
	var c EncryptedContent
	err := db.conn.QueryRow(
		`SELECT id, source_type, source_id, iv, ciphertext, created_at
		 FROM encrypted_content WHERE id = ?`, id,
	).Scan(&c.ID, &c.SourceType, &c.SourceID, &c.IV, &c.Ciphertext, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (db *DB) deleteContentBySourceTx(tx *sql.Tx, sourceType, sourceID string) error {
	_, err := tx.Exec(
		`DELETE FROM encrypted_content WHERE source_type = ? AND source_id = ?`,
		sourceType, sourceID,
	)
	return err
}

func NewContentID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("cnt_%s_%d", hex.EncodeToString(b), time.Now().UnixNano())
}

func (db *DB) ListAllContent() ([]EncryptedContent, error) {
	rows, err := db.conn.Query(
		`SELECT id, source_type, source_id, iv, ciphertext, created_at FROM encrypted_content`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var contents []EncryptedContent
	for rows.Next() {
		var c EncryptedContent
		if err := rows.Scan(&c.ID, &c.SourceType, &c.SourceID, &c.IV, &c.Ciphertext, &c.CreatedAt); err != nil {
			return nil, err
		}
		contents = append(contents, c)
	}
	return contents, rows.Err()
}
