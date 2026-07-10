package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

const exportSchemaVersion = 1

type exportDocument struct {
	SchemaVersion int           `json:"schema_version"`
	ExportedAt    time.Time     `json:"exported_at"`
	Entries       []entry.Entry `json:"entries"`
}

// ExportJSON writes all entries and the schema version to w.
func (s *Store) ExportJSON(w io.Writer) error {
	rows, err := s.db.Query(selectCols + ` ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("store: export list: %w", err)
	}
	defer rows.Close()

	var entries []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return fmt.Errorf("store: export scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: export rows: %w", err)
	}

	doc := exportDocument{
		SchemaVersion: exportSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Entries:       entries,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("store: export encode: %w", err)
	}
	return nil
}

// ImportMode controls how ImportJSON merges data.
type ImportMode string

const (
	ImportMerge   ImportMode = "merge"
	ImportReplace ImportMode = "replace"
)

// ImportJSON loads entries from r. merge skips existing IDs; replace deletes all
// entries first.
func (s *Store) ImportJSON(r io.Reader, mode ImportMode) (int, error) {
	var doc exportDocument
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return 0, fmt.Errorf("store: import decode: %w", err)
	}
	if doc.SchemaVersion == 0 {
		return 0, fmt.Errorf("store: import: missing schema_version")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("store: import begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if mode == ImportReplace {
		if _, err := tx.Exec(`DELETE FROM entries`); err != nil {
			return 0, fmt.Errorf("store: import clear: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM entry_history`); err != nil {
			return 0, fmt.Errorf("store: import clear history: %w", err)
		}
	}

	imported := 0
	for _, e := range doc.Entries {
		if e.ID == "" {
			continue
		}
		if mode == ImportMerge {
			var exists int
			err := tx.QueryRow(`SELECT 1 FROM entries WHERE id = ?`, e.ID).Scan(&exists)
			if err == nil {
				continue
			}
			if err != sql.ErrNoRows {
				return imported, fmt.Errorf("store: import lookup: %w", err)
			}
		}
		if err := insertEntry(tx, e, string(e.Status), e.ProposedAt); err != nil {
			return imported, fmt.Errorf("store: import entry %q: %w", e.Name, err)
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		return imported, fmt.Errorf("store: import commit: %w", err)
	}
	return imported, nil
}
