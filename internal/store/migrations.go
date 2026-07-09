package store

import (
	"database/sql"
	"fmt"
)

type migration struct {
	name string
	fn   func(*sql.Tx) error
}

var migrations = []migration{
	{name: "add delivery and grp columns", fn: migrateV1DeliveryGrp},
	{name: "add entry indices", fn: migrateV2Indices},
	{name: "unique active entry names", fn: migrateV3UniqueActiveName},
	{name: "entry history table", fn: migrateV4EntryHistory},
}

func runMigrations(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		m := migrations[i]
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d begin: %w", i+1, err)
		}
		if err := m.fn(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d (%s): %w", i+1, m.name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d set user_version: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d commit: %w", i+1, err)
		}
	}
	return nil
}

func migrateV1DeliveryGrp(tx *sql.Tx) error {
	if !hasColumnTx(tx, "entries", "delivery") {
		if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN delivery TEXT NOT NULL DEFAULT 'init'`); err != nil {
			return fmt.Errorf("add delivery column: %w", err)
		}
	}
	if !hasColumnTx(tx, "entries", "grp") {
		if _, err := tx.Exec(`ALTER TABLE entries ADD COLUMN grp TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add grp column: %w", err)
		}
	}
	return nil
}

func migrateV2Indices(tx *sql.Tx) error {
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_entries_status ON entries(status)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_kind ON entries(kind)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_name ON entries(name)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_target ON entries(target_id)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateV3UniqueActiveName(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT name, COUNT(*) AS cnt
		FROM entries
		WHERE status = 'active'
		GROUP BY name
		HAVING cnt > 1
	`)
	if err != nil {
		return fmt.Errorf("find duplicate active names: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return err
		}
		var ids []string
		idRows, err := tx.Query(`
			SELECT id FROM entries
			WHERE status = 'active' AND name = ?
			ORDER BY created_at, id
		`, name)
		if err != nil {
			return err
		}
		for idRows.Next() {
			var id string
			if err := idRows.Scan(&id); err != nil {
				idRows.Close()
				return err
			}
			ids = append(ids, id)
		}
		idRows.Close()
		if err := idRows.Err(); err != nil {
			return err
		}
		for i := 1; i < len(ids); i++ {
			newName := fmt.Sprintf("%s (%d)", name, i+1)
			if _, err := tx.Exec(`UPDATE entries SET name = ? WHERE id = ?`, newName, ids[i]); err != nil {
				return fmt.Errorf("rename duplicate %q: %w", name, err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_active_name
		ON entries(name) WHERE status = 'active'
	`)
	return err
}

func migrateV4EntryHistory(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_history (
			id           TEXT PRIMARY KEY,
			entry_id     TEXT NOT NULL,
			snapshot_json TEXT NOT NULL,
			action       TEXT NOT NULL,
			actor        TEXT NOT NULL DEFAULT '',
			created_at   DATETIME NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_entry_history_entry_id ON entry_history(entry_id);
	`)
	return err
}

func hasColumnTx(tx *sql.Tx, table, column string) bool {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name, typ string
			notNull   int
			dflt      sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}
