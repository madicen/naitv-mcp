package store

import (
	"fmt"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func (s *Store) IntegrityCheck() error {
	var result string
	if err := s.db.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("%s", result)
	}
	return nil
}

func (s *Store) FTSOutOfSync() (bool, error) {
	var ftsCount, entryCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM entries_fts`).Scan(&ftsCount); err != nil {
		return false, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM entries WHERE status != 'archived'`).Scan(&entryCount); err != nil {
		return false, err
	}
	return ftsCount != entryCount, nil
}

func (s *Store) RebuildFTS() error {
	_, err := s.db.Exec(`INSERT INTO entries_fts(entries_fts) VALUES('rebuild')`)
	return err
}

func (s *Store) OrphanProposals() ([]entry.Entry, error) {
	rows, err := s.db.Query(`
		SELECT ` + selectColsNoPrefix + `
		FROM entries e
		WHERE e.status = 'pending' AND e.target_id != ''
		  AND NOT EXISTS (SELECT 1 FROM entries t WHERE t.id = e.target_id)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ListArchived(kind string) ([]entry.Entry, error) {
	query := selectCols + ` WHERE status = 'archived'`
	args := []any{}
	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list archived: %w", err)
	}
	defer rows.Close()
	var results []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

func (s *Store) Restore(id string) error {
	e, err := s.Get(id)
	if err != nil {
		return err
	}
	if e.Status != entry.StatusArchived {
		return fmt.Errorf("store: restore: entry %s is not archived", id)
	}
	exists, err := s.activeNameExists(e.Name, id)
	if err != nil {
		return err
	}
	if exists {
		return ErrNameConflict
	}
	if err := s.recordHistory(e, "restore", ""); err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE entries SET status='active', updated_at=? WHERE id=?`, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: restore: not found: %s", id)
	}
	return nil
}

func (s *Store) RecordAccess(id string) error {
	_, err := s.db.Exec(`UPDATE entries SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`, time.Now().UTC(), id)
	return err
}

func (s *Store) RecordAccessBatch(ids []string) error {
	now := time.Now().UTC()
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := s.db.Exec(`UPDATE entries SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`, now, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) StaleEntries(staleDays, limit int) ([]entry.Entry, error) {
	if staleDays < 1 {
		staleDays = 90
	}
	if limit < 1 {
		limit = 5
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -staleDays)
	rows, err := s.db.Query(`
		SELECT `+selectColsNoPrefix+`
		FROM entries e
		WHERE e.status = 'active'
		  AND e.updated_at < ?
		  AND (e.last_accessed_at IS NULL OR e.access_count = 0)
		ORDER BY e.updated_at ASC
		LIMIT ?
	`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
