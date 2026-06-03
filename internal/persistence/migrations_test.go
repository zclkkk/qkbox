package persistence

import "testing"

func TestRuntimeStateOwnsActiveSnapshotSchema(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

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
		if name == "active_snapshot_id" {
			t.Fatal("profiles table must not own active_snapshot_id")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	var stateRows int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM runtime_state WHERE id = 1`).Scan(&stateRows); err != nil {
		t.Fatal(err)
	}
	if stateRows != 1 {
		t.Fatalf("runtime_state singleton rows = %d", stateRows)
	}
}
