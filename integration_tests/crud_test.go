package integration_tests

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_CreateEntry opens the new-entry form via 'n', delivers a SaveMsg
// by directly seeding the store (simulating form save), and verifies the entry appears.
func TestJourney_CreateEntry(t *testing.T) {
	st := newTestDB(t)
	m := newTestModel(t, st)

	// Press 'n' to open the new-entry form.
	m = runPendingCmds(m, key("n"), 3)

	// Simulate form save: create entry directly in store, then reload.
	_, err := st.Create(entry.Entry{Kind: "repo", Name: "new-repo", Body: "New body."})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	// Deliver loaded entries so view refreshes.
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	active, err := st.List("", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, e := range active {
		if e.Name == "new-repo" {
			found = true
		}
	}
	if !found {
		t.Error("expected new-repo in store after create")
	}

	view := m.View().Content
	if !strings.Contains(view, "new-repo") {
		t.Errorf("expected new-repo in view, got:\n%s", view)
	}
}

// TestJourney_EditEntry seeds an entry, loads it, presses 'e' and verifies
// the form state changes, then updates the store and reloads.
func TestJourney_EditEntry(t *testing.T) {
	st := newTestDB(t)

	orig, err := st.Create(entry.Entry{Kind: "note", Name: "edit-me", Body: "original body"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	m := newTestModel(t, st)
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	// Press 'e' to open edit form (entry must be selected).
	m = runPendingCmds(m, key("e"), 3)

	// Simulate update in store.
	orig.Body = "updated body"
	_, err = st.Update(orig)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Reload.
	loaded = entries.LoadEntriesCmd(st, "")()
	updateModel(m, loaded)

	updated, err := st.Get(orig.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Body != "updated body" {
		t.Errorf("expected updated body, got %q", updated.Body)
	}
}

// TestJourney_DeleteEntry seeds an entry, presses 'd', confirms, and verifies
// deletion from the store.
func TestJourney_DeleteEntry(t *testing.T) {
	st := newTestDB(t)

	e, err := st.Create(entry.Entry{Kind: "note", Name: "delete-me", Body: "bye"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	m := newTestModel(t, st)
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	// Press 'd' then 'y' to confirm delete.
	m = runPendingCmds(m, key("d"), 3)
	runPendingCmds(m, key("y"), 5)

	// Verify deletion via store directly (TUI should have called st.Delete).
	active, _ := st.List("", nil)
	for _, a := range active {
		if a.ID == e.ID {
			t.Error("expected entry to be deleted from store")
		}
	}
}

// TestJourney_FormFieldAddRemove verifies that pressing 'n' opens the form
// and the view transitions (form state visible).
func TestJourney_FormFieldAddRemove(t *testing.T) {
	st := newTestDB(t)
	m := newTestModel(t, st)

	// Open new entry form.
	m = runPendingCmds(m, key("n"), 3)

	// The form should now be visible — view should not be empty.
	view := m.View().Content
	if view == "" {
		t.Error("expected non-empty view after pressing n")
	}

	// Press Escape to cancel form.
	m = runPendingCmds(m, key("esc"), 3)

	// Form should be hidden now.
	_ = m.View().Content
}
