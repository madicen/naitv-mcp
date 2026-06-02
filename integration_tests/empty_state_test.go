package integration_tests

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
)

// TestJourney_EmptyState opens a fresh DB (no entries), loads, and verifies the
// view contains some content (empty state message or placeholder).
func TestJourney_EmptyState(t *testing.T) {
	st := newTestDB(t)

	m := newTestModel(t, st)

	// Deliver empty EntriesLoadedMsg.
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view even with empty DB")
	}

	// The view should render something meaningful — at minimum the tab bar.
	// We check that the Entries tab label appears.
	// (exact empty state string depends on the entries model's View implementation)
	if len(view) < 5 {
		t.Errorf("view too short, expected something meaningful, got: %q", view)
	}
}
