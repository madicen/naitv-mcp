package store_test

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestStore_HealthHelpers(t *testing.T) {
	s := openStore(t)
	if err := s.IntegrityCheck(); err != nil {
		t.Fatalf("IntegrityCheck: %v", err)
	}
	outOfSync, err := s.FTSOutOfSync()
	if err != nil {
		t.Fatalf("FTSOutOfSync: %v", err)
	}
	if outOfSync {
		t.Fatal("expected FTS in sync on fresh DB")
	}
	if orphans, err := s.OrphanProposals(); err != nil || len(orphans) != 0 {
		t.Fatalf("OrphanProposals = %v, %v", orphans, err)
	}
}

func TestStore_AccessAndStale(t *testing.T) {
	s := openStore(t)
	created, err := s.Create(entry.Entry{Kind: "note", Name: "stale-note", Body: "old"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.RecordAccess(created.ID); err != nil {
		t.Fatalf("RecordAccess: %v", err)
	}
	got, err := s.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessCount != 1 || got.LastAccessedAt == nil {
		t.Fatalf("access telemetry not recorded: %#v", got)
	}
	if err := s.RecordAccessBatch([]string{created.ID, ""}); err != nil {
		t.Fatalf("RecordAccessBatch: %v", err)
	}
	got, _ = s.Get(created.ID)
	if got.AccessCount != 2 {
		t.Fatalf("AccessCount = %d", got.AccessCount)
	}
}

func TestStore_ArchiveRestore(t *testing.T) {
	s := openStore(t)
	created, err := s.Create(entry.Entry{Kind: "note", Name: "archive-me", Body: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	archived, err := s.ListArchived("")
	if err != nil || len(archived) != 1 {
		t.Fatalf("ListArchived = %#v, %v", archived, err)
	}
	if err := s.Restore(created.ID); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got, err := s.Get(created.ID)
	if err != nil || got.Status != entry.StatusActive {
		t.Fatalf("after restore: %#v, %v", got, err)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
