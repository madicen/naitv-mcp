package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

// HistoryRecord is one snapshot of an entry at a point in time.
type HistoryRecord struct {
	ID        string
	EntryID   string
	Snapshot  entry.Entry
	Action    string
	Actor     string
	CreatedAt time.Time
}

func (s *Store) recordHistory(e entry.Entry, action, actor string) error {
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO entry_history (id, entry_id, snapshot_json, action, actor, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newID(), e.ID, string(b), action, actor, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

// History returns snapshots for an entry, newest first.
func (s *Store) History(entryID string) ([]HistoryRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, entry_id, snapshot_json, action, actor, created_at
		FROM entry_history
		WHERE entry_id = ?
		ORDER BY created_at DESC, id DESC
	`, entryID)
	if err != nil {
		return nil, fmt.Errorf("store: history: %w", err)
	}
	defer rows.Close()

	var records []HistoryRecord
	for rows.Next() {
		var rec HistoryRecord
		var snapshotJSON string
		if err := rows.Scan(&rec.ID, &rec.EntryID, &snapshotJSON, &rec.Action, &rec.Actor, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: history scan: %w", err)
		}
		if err := json.Unmarshal([]byte(snapshotJSON), &rec.Snapshot); err != nil {
			return nil, fmt.Errorf("store: history unmarshal: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: history rows: %w", err)
	}
	return records, nil
}

// RestoreVersion replaces the current entry with a historical snapshot.
func (s *Store) RestoreVersion(historyID string) (entry.Entry, error) {
	var snapshotJSON string
	var entryID string
	err := s.db.QueryRow(`
		SELECT entry_id, snapshot_json FROM entry_history WHERE id = ?
	`, historyID).Scan(&entryID, &snapshotJSON)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: restore version: %w", err)
	}

	var snap entry.Entry
	if err := json.Unmarshal([]byte(snapshotJSON), &snap); err != nil {
		return entry.Entry{}, fmt.Errorf("store: restore unmarshal: %w", err)
	}

	current, err := s.Get(entryID)
	if err != nil {
		return entry.Entry{}, err
	}

	snap.ID = current.ID
	snap.CreatedAt = current.CreatedAt
	snap.Status = current.Status
	snap.ProposedBy = current.ProposedBy
	snap.ProposedAt = current.ProposedAt
	snap.TargetID = current.TargetID

	if err := s.recordHistory(current, "restore", ""); err != nil {
		return entry.Entry{}, err
	}

	return s.Update(snap)
}
