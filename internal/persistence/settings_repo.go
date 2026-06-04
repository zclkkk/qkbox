package persistence

import "database/sql"

func (db *DB) SetSetting(key string, value []byte) error {
	_, err := db.conn.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (db *DB) GetSetting(key string) ([]byte, error) {
	var value []byte
	err := db.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (db *DB) DeleteSetting(key string) error {
	_, err := db.conn.Exec(`DELETE FROM settings WHERE key = ?`, key)
	return err
}
