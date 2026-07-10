package integration_tests

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_BrowseEntries seeds entries, loads them, navigates the list, and
// verifies the view reflects the loaded content.
func TestJourney_BrowseEntries(t *testing.T) {
	st := newTestDB(t)

	// Seed a couple of entries.
	_, err := st.Create(entry.Entry{Kind: "repo", Name: "my-repo", Body: "A great repo."})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	_, err = st.Create(entry.Entry{Kind: "note", Name: "my-note", Body: "An important note."})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	m := newTestModel(t, st)

	// Manually deliver an EntriesLoadedMsg so the model has entries.
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	view := m.View().Content
	if !strings.Contains(view, "my-repo") && !strings.Contains(view, "my-note") {
		t.Errorf("expected entries in view, got:\n%s", view)
	}

	// Navigate down.
	m = updateModel(m, keyType(tea.KeyDown))
	_ = m.View().Content // should not panic
}

// TestJourney_KindTabsAutoPopulate creates entries of distinct kinds and verifies
// the kinds appear in the view after loading.
func TestJourney_KindTabsAutoPopulate(t *testing.T) {
	st := newTestDB(t)

	kinds := []string{"repo", "note", "workflow", "fact"}
	for _, k := range kinds {
		_, err := st.Create(entry.Entry{Kind: k, Name: "test-" + k, Body: "body"})
		if err != nil {
			t.Fatalf("create %s: %v", k, err)
		}
	}

	m := newTestModel(t, st)

	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	view := m.View().Content
	for _, k := range kinds {
		if !strings.Contains(view, k) {
			t.Errorf("expected kind %q in view, got:\n%s", k, view)
		}
	}
}
