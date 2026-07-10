package store_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestStore_NameConflict(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.Create(makeEntry("note", "dup", "one")); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	_, err := s.Create(makeEntry("note", "dup", "two"))
	if !errors.Is(err, store.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}
}

func TestStore_ApproveAllRollback(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.Create(makeEntry("note", "existing", "body")); err != nil {
		t.Fatalf("Create existing: %v", err)
	}

	p1, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "other", Body: "body"})
	p2, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "existing", Body: "conflict"})

	_, err := s.ApproveAll()
	if err == nil {
		t.Fatal("expected ApproveAll to fail on name conflict")
	}

	if _, err := s.Get(p1.ID); err != nil {
		t.Fatalf("p1 should still exist after rollback: %v", err)
	}
	if _, err := s.Get(p2.ID); err != nil {
		t.Fatalf("p2 should still exist after rollback: %v", err)
	}

	count, _ := s.PendingCount()
	if count != 2 {
		t.Errorf("expected 2 pending after rollback, got %d", count)
	}

	all, _ := s.List("", nil)
	if len(all) != 1 {
		t.Errorf("expected only original active entry, got %d", len(all))
	}
}

func TestStore_HistoryOnUpdate(t *testing.T) {
	s := openTestStore(t)

	e, err := s.Create(makeEntry("note", "hist", "v1"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	e.Body = "v2"
	if _, err := s.Update(e); err != nil {
		t.Fatalf("Update: %v", err)
	}

	records, err := s.History(e.ID)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(records))
	}
	if records[0].Snapshot.Body != "v1" {
		t.Errorf("history snapshot body: got %q, want v1", records[0].Snapshot.Body)
	}
}

func TestStore_ExportImportRoundTrip(t *testing.T) {
	s := openTestStore(t)

	e, err := s.Create(entry.Entry{
		Kind: "note",
		Name: "export-me",
		Body: "payload",
		Tags: []string{"a"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var buf bytes.Buffer
	if err := s.ExportJSON(&buf); err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	if !strings.Contains(buf.String(), "export-me") {
		t.Fatalf("export missing entry: %s", buf.String())
	}

	s2 := openTestStore(t)
	n, err := s2.ImportJSON(&buf, store.ImportReplace)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported %d entries, want 1", n)
	}

	got, err := s2.Get(e.ID)
	if err != nil {
		t.Fatalf("Get imported: %v", err)
	}
	if got.Name != "export-me" || got.Body != "payload" {
		t.Errorf("import mismatch: %+v", got)
	}
}

func TestStore_SoftDeleteExcludedFromSearch(t *testing.T) {
	s := openTestStore(t)

	e, err := s.Create(makeEntry("note", "findme", "unique searchable phrase"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	results, err := s.Search("unique")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected archived entry excluded from search, got %d", len(results))
	}
}
