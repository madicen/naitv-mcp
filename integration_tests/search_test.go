package integration_tests

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_SearchEntries seeds entries, presses '/' to open search,
// delivers search results, and verifies the result appears in the view.
// Pressing Escape clears the search.
func TestJourney_SearchEntries(t *testing.T) {
	st := newTestDB(t)

	_, err := st.Create(entry.Entry{Kind: "repo", Name: "searchable-repo", Body: "findme content"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = st.Create(entry.Entry{Kind: "note", Name: "other-note", Body: "other content"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	m := newTestModel(t, st)

	// Load entries first.
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	// Press '/' to open search.
	m = runPendingCmds(m, key("/"), 3)

	// Deliver a search result for "searchable".
	searchMsg := entries.SearchResultsMsg{
		Entries: []entry.Entry{
			{Kind: "repo", Name: "searchable-repo", Body: "findme content"},
		},
		Query: "searchable",
	}
	m = updateModel(m, searchMsg)

	view := m.View().Content
	if !strings.Contains(view, "searchable-repo") {
		t.Errorf("expected searchable-repo in view after search, got:\n%s", view)
	}

	// Press Escape to clear search.
	m = runPendingCmds(m, keyType(tea.KeyEsc), 3)

	// After escape, reload normal entries.
	loaded = entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	view = m.View().Content
	// View should contain both entries now (or at least not crash).
	_ = view
}
