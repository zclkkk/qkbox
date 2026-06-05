package persistence

import "fmt"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		draft_content_id TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS encrypted_content (
		id TEXT PRIMARY KEY,
		source_type TEXT NOT NULL,
		source_id TEXT NOT NULL,
		iv BLOB NOT NULL,
		ciphertext BLOB NOT NULL,
		created_at INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS snapshots (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		content_id TEXT NOT NULL REFERENCES encrypted_content(id),
		validation_status TEXT NOT NULL,
		diagnostics_json BLOB,
		runtime_summary_json BLOB,
		required_capabilities_json BLOB,
		created_at INTEGER NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value BLOB NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS runtime_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		active_snapshot_id TEXT REFERENCES snapshots(id) ON DELETE SET NULL,
		updated_at INTEGER NOT NULL
	);`,

	`INSERT INTO runtime_state (id, active_snapshot_id, updated_at)
	 SELECT 1, NULL, 0
	 WHERE NOT EXISTS (SELECT 1 FROM runtime_state WHERE id = 1);`,

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
