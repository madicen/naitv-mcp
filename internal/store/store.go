package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS entries (
    id           TEXT PRIMARY KEY,
    kind         TEXT NOT NULL,
    name         TEXT NOT NULL,
    grp          TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    tags         TEXT NOT NULL DEFAULT '[]',
    fields       TEXT NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'active',
    delivery     TEXT NOT NULL DEFAULT 'init',
    proposed_by  TEXT NOT NULL DEFAULT '',
    proposed_at  DATETIME,
    target_id    TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    name, body, tags, fields,
    content='entries', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
    INSERT INTO entries_fts(rowid, name, body, tags, fields)
    VALUES (new.rowid, new.name, new.body, new.tags, new.fields);
END;

CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, name, body, tags, fields)
    VALUES ('delete', old.rowid, old.name, old.body, old.tags, old.fields);
    INSERT INTO entries_fts(rowid, name, body, tags, fields)
    VALUES (new.rowid, new.name, new.body, new.tags, new.fields);
END;

CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, name, body, tags, fields)
    VALUES ('delete', old.rowid, old.name, old.body, old.tags, old.fields);
END;
`

// Store wraps a SQLite database for single-goroutine use.
type Store struct {
	db     *sql.DB
	dbPath string

	onChangeMu sync.Mutex
	onChange   []func()
}

// DefaultDBPath returns ~/.config/naitv-mcp/context.db, or the value of
// NAITV_MCP_DB if set.
func DefaultDBPath() string {
	if p := os.Getenv("NAITV_MCP_DB"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "naitv-mcp", "context.db")
}

// Open opens (or creates) the SQLite database at dbPath and applies the schema.
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("store: create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}
	return &Store{db: db, dbPath: dbPath}, nil
}

// DBPath returns the filesystem path of the SQLite database.
func (s *Store) DBPath() string {
	return s.dbPath
}

// ModTime returns the last modification time of the database file.
func (s *Store) ModTime() (time.Time, error) {
	if s.dbPath == "" {
		return time.Time{}, nil
	}
	info, err := os.Stat(s.dbPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// OnChange registers fn to be called after mutating store operations.
func (s *Store) OnChange(fn func()) {
	s.onChangeMu.Lock()
	defer s.onChangeMu.Unlock()
	s.onChange = append(s.onChange, fn)
}

func (s *Store) notifyChange() {
	s.onChangeMu.Lock()
	fns := append([]func(){}, s.onChange...)
	s.onChangeMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

const insertEntrySQL = `INSERT INTO entries (id, kind, name, grp, body, tags, fields, status, delivery, proposed_by, proposed_at, target_id, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func insertEntry(db execer, e entry.Entry, status string, proposedAt *time.Time) error {
	tagsJSON, err := marshalTags(e.Tags)
	if err != nil {
		return err
	}
	fieldsJSON, err := marshalFields(e.Fields)
	if err != nil {
		return err
	}
	var proposedAtArg interface{}
	if proposedAt != nil {
		proposedAtArg = proposedAt.UTC()
	}
	_, err = db.Exec(
		insertEntrySQL,
		e.ID, e.Kind, e.Name, e.Group, e.Body,
		tagsJSON, fieldsJSON,
		status, string(e.DeliveryOrDefault()),
		e.ProposedBy, proposedAtArg, e.TargetID,
		e.CreatedAt, e.UpdatedAt,
	)
	return err
}

func (s *Store) activeNameExists(name, excludeID string) (bool, error) {
	query := `SELECT 1 FROM entries WHERE status = 'active' AND name = ?`
	args := []any{name}
	if excludeID != "" {
		query += ` AND id != ?`
		args = append(args, excludeID)
	}
	var one int
	err := s.db.QueryRow(query, args...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// newID generates a new ULID string.
func newID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// marshalTags encodes a []string as JSON.
func marshalTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}
	return string(b), nil
}

// marshalFields encodes a map[string]string as JSON.
func marshalFields(fields map[string]string) (string, error) {
	if fields == nil {
		fields = map[string]string{}
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("marshal fields: %w", err)
	}
	return string(b), nil
}

// scanEntry reads a row produced by selectCols into an Entry.
func scanEntry(row interface {
	Scan(...any) error
}) (entry.Entry, error) {
	var e entry.Entry
	var tagsJSON, fieldsJSON string
	var proposedAt sql.NullTime
	var status, delivery string

	err := row.Scan(
		&e.ID, &e.Kind, &e.Name, &e.Group, &e.Body,
		&tagsJSON, &fieldsJSON,
		&status, &delivery,
		&e.ProposedBy, &proposedAt, &e.TargetID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return entry.Entry{}, err
	}

	e.Status = entry.Status(status)
	e.Delivery = entry.Delivery(delivery)

	if err := json.Unmarshal([]byte(tagsJSON), &e.Tags); err != nil {
		e.Tags = []string{}
	}
	if e.Tags == nil {
		e.Tags = []string{}
	}

	if err := json.Unmarshal([]byte(fieldsJSON), &e.Fields); err != nil {
		e.Fields = map[string]string{}
	}
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}

	if proposedAt.Valid {
		t := proposedAt.Time
		e.ProposedAt = &t
	}

	return e, nil
}

const selectCols = `SELECT id, kind, name, grp, body, tags, fields, status, delivery, proposed_by, proposed_at, target_id, created_at, updated_at FROM entries`

const selectColsNoPrefix = `e.id, e.kind, e.name, e.grp, e.body, e.tags, e.fields, e.status, e.delivery, e.proposed_by, e.proposed_at, e.target_id, e.created_at, e.updated_at`

// List returns active entries, optionally filtered by kind and/or tags.
// An empty kind means "all kinds". Tags is an AND filter.
func (s *Store) List(kind string, tags []string) ([]entry.Entry, error) {
	query := selectCols + ` WHERE status = 'active'`
	args := []any{}

	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	for _, tag := range tags {
		query += ` AND EXISTS (SELECT 1 FROM json_each(entries.tags) WHERE json_each.value = ?)`
		args = append(args, tag)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list: %w", err)
	}
	defer rows.Close()

	var results []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("store: list scan: %w", err)
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list rows: %w", err)
	}
	return results, nil
}

// Get retrieves an entry by ID (any status).
func (s *Store) Get(id string) (entry.Entry, error) {
	row := s.db.QueryRow(selectCols+` WHERE id = ?`, id)
	e, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return entry.Entry{}, fmt.Errorf("store: get: not found: %s", id)
	}
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: get: %w", err)
	}
	return e, nil
}

// GetByName retrieves an active entry by name.
func (s *Store) GetByName(name string) (entry.Entry, error) {
	row := s.db.QueryRow(selectCols+` WHERE name = ? AND status = 'active' LIMIT 1`, name)
	e, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return entry.Entry{}, fmt.Errorf("store: get by name: not found: %s", name)
	}
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: get by name: %w", err)
	}
	return e, nil
}

// Search performs a full-text search over active entries.
func (s *Store) Search(query string) ([]entry.Entry, error) {
	q := `
		SELECT ` + selectColsNoPrefix + `
		FROM entries_fts
		JOIN entries e ON entries_fts.rowid = e.rowid
		WHERE entries_fts MATCH ?
		  AND e.status = 'active'
		ORDER BY rank
	`
	rows, err := s.db.Query(q, query)
	if err != nil {
		return nil, fmt.Errorf("store: search: %w", err)
	}
	defer rows.Close()

	var results []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("store: search scan: %w", err)
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: search rows: %w", err)
	}
	return results, nil
}

// Create inserts a new active entry, generating a ULID id and timestamps.
func (s *Store) Create(e entry.Entry) (entry.Entry, error) {
	exists, err := s.activeNameExists(e.Name, "")
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: create: %w", err)
	}
	if exists {
		return entry.Entry{}, ErrNameConflict
	}

	now := time.Now().UTC()
	e.ID = newID()
	e.Status = entry.StatusActive
	e.Delivery = e.DeliveryOrDefault()
	e.CreatedAt = now
	e.UpdatedAt = now
	e.ProposedAt = nil

	if err := insertEntry(s.db, e, string(e.Status), nil); err != nil {
		return entry.Entry{}, fmt.Errorf("store: create: %w", err)
	}
	s.notifyChange()
	return e, nil
}

// Update saves changes to an existing entry, updating updated_at.
func (s *Store) Update(e entry.Entry) (entry.Entry, error) {
	if e.Status == entry.StatusActive {
		exists, err := s.activeNameExists(e.Name, e.ID)
		if err != nil {
			return entry.Entry{}, fmt.Errorf("store: update: %w", err)
		}
		if exists {
			return entry.Entry{}, ErrNameConflict
		}
	}

	before, err := s.Get(e.ID)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: update: %w", err)
	}
	if err := s.recordHistory(before, "update", ""); err != nil {
		return entry.Entry{}, err
	}

	e.UpdatedAt = time.Now().UTC()

	var proposedAt interface{}
	if e.ProposedAt != nil {
		proposedAt = e.ProposedAt.UTC()
	}

	tagsJSON, err := marshalTags(e.Tags)
	if err != nil {
		return entry.Entry{}, err
	}
	fieldsJSON, err := marshalFields(e.Fields)
	if err != nil {
		return entry.Entry{}, err
	}

	res, err := s.db.Exec(
		`UPDATE entries SET kind=?, name=?, grp=?, body=?, tags=?, fields=?, status=?, delivery=?, proposed_by=?, proposed_at=?, target_id=?, updated_at=? WHERE id=?`,
		e.Kind, e.Name, e.Group, e.Body,
		tagsJSON, fieldsJSON,
		string(e.Status), string(e.DeliveryOrDefault()),
		e.ProposedBy, proposedAt, e.TargetID,
		e.UpdatedAt, e.ID,
	)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return entry.Entry{}, fmt.Errorf("store: update: not found: %s", e.ID)
	}
	s.notifyChange()
	return e, nil
}

// SetDelivery updates only the delivery mode of an entry, leaving other fields
// untouched. updated_at is bumped.
func (s *Store) SetDelivery(id string, d entry.Delivery) error {
	if d == "" {
		d = entry.DeliveryInit
	}
	before, err := s.Get(id)
	if err != nil {
		return fmt.Errorf("store: set delivery: %w", err)
	}
	if err := s.recordHistory(before, "set_delivery", ""); err != nil {
		return err
	}

	res, err := s.db.Exec(
		`UPDATE entries SET delivery=?, updated_at=? WHERE id=?`,
		string(d), time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("store: set delivery: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: set delivery: not found: %s", id)
	}
	s.notifyChange()
	return nil
}

// Delete archives an entry by ID (soft delete).
func (s *Store) Delete(id string) error {
	e, err := s.Get(id)
	if err != nil {
		return fmt.Errorf("store: delete: %w", err)
	}
	if err := s.recordHistory(e, "delete", ""); err != nil {
		return err
	}

	res, err := s.db.Exec(
		`UPDATE entries SET status='archived', updated_at=? WHERE id=?`,
		time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("store: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: delete: not found: %s", id)
	}
	s.notifyChange()
	return nil
}

// Purge permanently removes an entry by ID.
func (s *Store) Purge(id string) error {
	res, err := s.db.Exec(`DELETE FROM entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: purge: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: purge: not found: %s", id)
	}
	_, _ = s.db.Exec(`DELETE FROM entry_history WHERE entry_id = ?`, id)
	s.notifyChange()
	return nil
}

// CreatePending inserts a proposal entry with status=pending.
func (s *Store) CreatePending(e entry.Entry) (entry.Entry, error) {
	now := time.Now().UTC()
	e.ID = newID()
	e.Status = entry.StatusPending
	e.Delivery = e.DeliveryOrDefault()
	e.CreatedAt = now
	e.UpdatedAt = now
	e.ProposedAt = &now

	if err := insertEntry(s.db, e, string(e.Status), e.ProposedAt); err != nil {
		return entry.Entry{}, fmt.Errorf("store: create pending: %w", err)
	}
	s.notifyChange()
	return e, nil
}

// ListPending returns all entries with status=pending.
func (s *Store) ListPending() ([]entry.Entry, error) {
	rows, err := s.db.Query(selectCols + ` WHERE status = 'pending'`)
	if err != nil {
		return nil, fmt.Errorf("store: list pending: %w", err)
	}
	defer rows.Close()

	var results []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("store: list pending scan: %w", err)
		}
		results = append(results, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list pending rows: %w", err)
	}
	return results, nil
}

// PendingCount returns the number of pending proposals.
func (s *Store) PendingCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM entries WHERE status = 'pending'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: pending count: %w", err)
	}
	return count, nil
}

// Approve promotes a pending proposal to active.
// If target_id == "": creates a new active entry from proposal fields.
// If target_id != "": merges non-empty proposal fields into the target entry, then deletes proposal.
func (s *Store) Approve(proposalID string) (entry.Entry, error) {
	e, err := s.approve(s.db, proposalID)
	if err != nil {
		return entry.Entry{}, err
	}
	s.notifyChange()
	return e, nil
}

func (s *Store) approve(db execer, proposalID string) (entry.Entry, error) {
	getter, ok := db.(interface {
		QueryRow(query string, args ...any) *sql.Row
	})
	if !ok {
		return entry.Entry{}, fmt.Errorf("store: approve: unsupported executor")
	}

	proposal, err := scanEntry(getter.QueryRow(selectCols+` WHERE id = ?`, proposalID))
	if err == sql.ErrNoRows {
		return entry.Entry{}, fmt.Errorf("store: approve: not found: %s", proposalID)
	}
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: approve: %w", err)
	}
	if proposal.Status != entry.StatusPending {
		return entry.Entry{}, fmt.Errorf("store: approve: entry %s is not pending", proposalID)
	}

	if proposal.TargetID == "" {
		exists, err := s.activeNameExists(proposal.Name, proposalID)
		if err != nil {
			return entry.Entry{}, fmt.Errorf("store: approve: %w", err)
		}
		if exists {
			return entry.Entry{}, ErrNameConflict
		}

		now := time.Now().UTC()
		res, err := db.Exec(
			`UPDATE entries SET status='active', proposed_at=NULL, updated_at=? WHERE id=?`,
			now, proposalID,
		)
		if err != nil {
			return entry.Entry{}, fmt.Errorf("store: approve: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return entry.Entry{}, fmt.Errorf("store: approve: proposal not found")
		}
		proposal.Status = entry.StatusActive
		proposal.ProposedAt = nil
		proposal.UpdatedAt = now
		return proposal, nil
	}

	target, err := scanEntry(getter.QueryRow(selectCols+` WHERE id = ?`, proposal.TargetID))
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: approve: target: %w", err)
	}

	if err := s.recordHistory(target, "approve", ""); err != nil {
		return entry.Entry{}, err
	}

	if proposal.Name != "" {
		target.Name = proposal.Name
	}
	if proposal.Body != "" {
		target.Body = proposal.Body
	}
	if proposal.Kind != "" {
		target.Kind = proposal.Kind
	}
	if proposal.Group != "" {
		target.Group = proposal.Group
	}
	if len(proposal.Tags) > 0 {
		target.Tags = proposal.Tags
	}
	if len(proposal.Fields) > 0 {
		if target.Fields == nil {
			target.Fields = map[string]string{}
		}
		for k, v := range proposal.Fields {
			if v != "" {
				target.Fields[k] = v
			}
		}
	}

	target.UpdatedAt = time.Now().UTC()
	tagsJSON, err := marshalTags(target.Tags)
	if err != nil {
		return entry.Entry{}, err
	}
	fieldsJSON, err := marshalFields(target.Fields)
	if err != nil {
		return entry.Entry{}, err
	}

	res, err := db.Exec(
		`UPDATE entries SET kind=?, name=?, grp=?, body=?, tags=?, fields=?, updated_at=? WHERE id=?`,
		target.Kind, target.Name, target.Group, target.Body,
		tagsJSON, fieldsJSON, target.UpdatedAt, target.ID,
	)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("store: approve: update target: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return entry.Entry{}, fmt.Errorf("store: approve: target not found")
	}

	if _, err := db.Exec(`DELETE FROM entries WHERE id = ?`, proposalID); err != nil {
		return entry.Entry{}, fmt.Errorf("store: approve: delete proposal: %w", err)
	}

	return target, nil
}

// Reject deletes a pending proposal without applying it.
func (s *Store) Reject(proposalID string) error {
	proposal, err := s.Get(proposalID)
	if err != nil {
		return fmt.Errorf("store: reject: %w", err)
	}
	if proposal.Status != entry.StatusPending {
		return fmt.Errorf("store: reject: entry %s is not pending", proposalID)
	}
	if err := s.Purge(proposalID); err != nil {
		return fmt.Errorf("store: reject: %w", err)
	}
	return nil
}

// ApproveAll approves every pending proposal in a single transaction.
func (s *Store) ApproveAll() ([]entry.Entry, error) {
	pending, err := s.ListPending()
	if err != nil {
		return nil, fmt.Errorf("store: approve all: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: approve all begin: %w", err)
	}
	defer tx.Rollback()

	var results []entry.Entry
	for _, p := range pending {
		approved, err := s.approve(tx, p.ID)
		if err != nil {
			return results, fmt.Errorf("store: approve all: %w", err)
		}
		results = append(results, approved)
	}

	if err := tx.Commit(); err != nil {
		return results, fmt.Errorf("store: approve all commit: %w", err)
	}
	s.notifyChange()
	return results, nil
}
