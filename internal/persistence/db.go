package persistence

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn     *sql.DB
	stateDir string
}

func Open(stateDir string) (*DB, error) {
	path := filepath.Join(stateDir, "qkbox.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	db := &DB{conn: conn, stateDir: stateDir}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) StateDir() string {
	return db.stateDir
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) WithTx(fn func(tx *sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
