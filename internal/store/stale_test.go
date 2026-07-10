package store

import (
	"testing"
	"time"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestStaleEntries(t *testing.T) {
	s, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	created, err := s.Create(entry.Entry{Kind: "note", Name: "stale-note", Body: "old"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	old := time.Now().UTC().AddDate(0, 0, -120)
	if _, err := s.db.Exec(`UPDATE entries SET updated_at = ?, access_count = 0, last_accessed_at = NULL WHERE id = ?`, old, created.ID); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	stale, err := s.StaleEntries(90, 5)
	if err != nil {
		t.Fatalf("StaleEntries: %v", err)
	}
	if len(stale) != 1 || stale[0].Name != "stale-note" {
		t.Fatalf("stale = %#v", stale)
	}
}
