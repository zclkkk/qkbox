package persistence

import (
	"database/sql"
	"fmt"
)

const legacyProfileSchemaMessage = "legacy qkbox profile schema detected; delete qkbox.db and restart qkbox. Pre-release snapshot/encrypted_content databases are not migrated."

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS profile_subscriptions (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		update_policy TEXT NOT NULL,
		last_status TEXT NOT NULL,
		last_error_code TEXT,
		last_error_message TEXT,
		last_checked_at INTEGER,
		last_updated_at INTEGER,
		content_sha256 TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS data_assets (
		id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		source_url TEXT NOT NULL,
		status TEXT NOT NULL,
		cache_key TEXT,
		version TEXT,
		content_sha256 TEXT,
		size_bytes INTEGER,
		last_error_code TEXT,
		last_error_message TEXT,
		last_checked_at INTEGER,
		last_updated_at INTEGER,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`,
}

func (db *DB) migrate() error {
	if err := db.rejectLegacyProfileSchema(); err != nil {
		return err
	}

	// bootstrap schema_version table
	if _, err := db.conn.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	if _, err := db.conn.Exec(`INSERT INTO schema_version (version) SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`); err != nil {
		return fmt.Errorf("init schema_version: %w", err)
	}

	var version int
	if err := db.conn.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version > len(migrations) {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, len(migrations))
	}

	for i, stmt := range migrations {
		if i < version {
			continue
		}
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}

	if _, err := db.conn.Exec("UPDATE schema_version SET version = ?", len(migrations)); err != nil {
		return fmt.Errorf("update schema version: %w", err)
	}
	return nil
}

func (db *DB) rejectLegacyProfileSchema() error {
	for _, table := range []string{"snapshots", "encrypted_content", "runtime_state"} {
		exists, err := db.tableExists(table)
		if err != nil {
			return fmt.Errorf("inspect legacy schema: %w", err)
		}
		if exists {
			return fmt.Errorf("%s Found legacy table %q.", legacyProfileSchemaMessage, table)
		}
	}

	exists, err := db.tableExists("profiles")
	if err != nil {
		return fmt.Errorf("inspect profiles schema: %w", err)
	}
	if !exists {
		return nil
	}
	hasDraft, err := db.tableHasColumn("profiles", "draft_content_id")
	if err != nil {
		return fmt.Errorf("inspect profiles columns: %w", err)
	}
	if hasDraft {
		return fmt.Errorf("%s Found legacy column %q on profiles.", legacyProfileSchemaMessage, "draft_content_id")
	}
	return nil
}

func (db *DB) tableExists(name string) (bool, error) {
	var found string
	err := db.conn.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		name,
	).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) tableHasColumn(tableName, columnName string) (bool, error) {
	rows, err := db.conn.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}
