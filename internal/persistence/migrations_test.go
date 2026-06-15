package persistence

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileConfigSchema(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, table := range []string{"profiles", "profile_subscriptions", "data_assets", "settings", "schema_version"} {
		if exists, err := db.tableExists(table); err != nil {
			t.Fatalf("inspect table %s: %v", table, err)
		} else if !exists {
			t.Fatalf("missing table %s", table)
		}
	}
	for _, table := range []string{"snapshots", "encrypted_content", "runtime_state"} {
		if exists, err := db.tableExists(table); err != nil {
			t.Fatalf("inspect table %s: %v", table, err)
		} else if exists {
			t.Fatalf("legacy table %s must not exist", table)
		}
	}
	if hasDraft, err := db.tableHasColumn("profiles", "draft_content_id"); err != nil {
		t.Fatal(err)
	} else if hasDraft {
		t.Fatal("profiles.draft_content_id must not exist")
	}
	if hasContent, err := db.tableHasColumn("profiles", "content"); err != nil {
		t.Fatal(err)
	} else if !hasContent {
		t.Fatal("profiles.content must exist")
	}

	var contentType string
	var contentNotNull int
	rows, err := db.conn.Query(`PRAGMA table_info(profiles)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "content" {
			contentType = strings.ToUpper(typ)
			contentNotNull = notNull
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if contentType != "TEXT" {
		t.Fatalf("profiles.content type = %s, want TEXT", contentType)
	}
	if contentNotNull != 1 {
		t.Fatalf("profiles.content NOT NULL = %d, want 1", contentNotNull)
	}
}

func TestSettingsAreTextBackedWithByteAPI(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	value := []byte(`{"owner":"qkbox","snapshot":{"raw":true}}`)
	if err := db.SetSetting("proxy_owner", value); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetSetting("proxy_owner")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatalf("setting = %q, want %q", got, value)
	}

	var sqliteType string
	if err := db.conn.QueryRow(`SELECT typeof(value) FROM settings WHERE key = 'proxy_owner'`).Scan(&sqliteType); err != nil {
		t.Fatal(err)
	}
	if sqliteType != "text" {
		t.Fatalf("settings.value sqlite type = %s, want text", sqliteType)
	}
}

func TestActiveProfileIDUsesSettings(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.SetActiveProfileTx(tx, "prf_active")
	}); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetActiveProfileID()
	if err != nil {
		t.Fatal(err)
	}
	if got != "prf_active" {
		t.Fatalf("active profile = %q, want prf_active", got)
	}
	var stored string
	if err := db.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, activeProfileSettingKey).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "prf_active" {
		t.Fatalf("settings active profile = %q, want prf_active", stored)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.ClearActiveProfileIfMatchesTx(tx, "prf_other")
	}); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetActiveProfileID()
	if err != nil {
		t.Fatal(err)
	}
	if got != "prf_active" {
		t.Fatalf("active profile after non-match clear = %q, want prf_active", got)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.ClearActiveProfileIfMatchesTx(tx, "prf_active")
	}); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetActiveProfileID()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("active profile after match clear = %q, want empty", got)
	}
}

func TestLegacyProfileSchemaFailsOpen(t *testing.T) {
	dir := t.TempDir()
	conn, err := sql.Open("sqlite", filepath.Join(dir, "qkbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE profiles (id TEXT PRIMARY KEY, name TEXT NOT NULL, draft_content_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE encrypted_content (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(dir)
	if db != nil {
		_ = db.Close()
	}
	if err == nil {
		t.Fatal("expected legacy schema error")
	}
	if !strings.Contains(err.Error(), "legacy qkbox profile schema detected") {
		t.Fatalf("error = %v", err)
	}
}

func TestAssetPlaneSchema(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, table := range []string{"profile_subscriptions", "data_assets"} {
		var name string
		if err := db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("missing table %s: %v", table, err)
		}
	}
}
