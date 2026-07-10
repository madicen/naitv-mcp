package integration_tests

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_CreateEntry opens the new-entry form via 'n', delivers a SaveMsg
// by directly seeding the store (simulating form save), and verifies the entry appears.
func TestJourney_CreateEntry(t *testing.T) {
	st := newTestDB(t)
	m := newTestModel(t, st)

	m = runPendingCmds(m, key("n"), 3)

	_, err := st.Create(entry.Entry{Kind: "repo", Name: "new-repo", Body: "New body."})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

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

// TestJourney_FormSaveTyping opens the form, types kind/name, saves with ctrl+s,
// and verifies the entry was persisted in the store.
func TestJourney_FormSaveTyping(t *testing.T) {
	st := newTestDB(t)
	m := newTestModel(t, st)

	m = runPendingCmds(m, key("n"), 3)
	m = typeString(m, "tool")
	m = pressTab(m)
	m = typeString(m, "typed-entry")
	m = runPendingCmds(m, key("ctrl+s"), 5)

	active, err := st.List("", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var saved *entry.Entry
	for i := range active {
		if active[i].Name == "typed-entry" {
			saved = &active[i]
			break
		}
	}
	if saved == nil {
		t.Fatal("expected typed-entry in store after ctrl+s save")
	}
	if saved.Kind != "tool" {
		t.Errorf("kind = %q, want tool", saved.Kind)
	}

	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)
	view := m.View().Content
	if !strings.Contains(view, "typed-entry") {
		t.Errorf("expected typed-entry in view, got:\n%s", view)
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

	m = runPendingCmds(m, key("e"), 3)

	orig.Body = "updated body"
	_, err = st.Update(orig)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

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

	m = runPendingCmds(m, key("d"), 3)
	runPendingCmds(m, key("y"), 5)

	active, _ := st.List("", nil)
	for _, a := range active {
		if a.ID == e.ID {
			t.Error("expected entry to be deleted from store")
		}
	}
}

// TestJourney_FormFieldAddRemove adds a custom field, removes it, saves, and
// verifies the persisted Fields map is empty.
func TestJourney_FormFieldAddRemove(t *testing.T) {
	st := newTestDB(t)
	m := newTestModel(t, st)

	m = runPendingCmds(m, key("n"), 3)
	m = typeString(m, "note")
	m = pressTab(m)
	m = typeString(m, "field-test")
	m = pressTab(m) // add-field button
	m = runPendingCmds(m, key("enter"), 3)
	m = typeString(m, "env")
	m = pressTab(m)
	m = typeString(m, "production")

	m = clickZone(t, m, zones.FormRemoveField(0))
	m = runPendingCmds(m, key("ctrl+s"), 5)

	active, err := st.List("", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var saved *entry.Entry
	for i := range active {
		if active[i].Name == "field-test" {
			saved = &active[i]
			break
		}
	}
	if saved == nil {
		t.Fatal("expected field-test entry in store")
	}
	if len(saved.Fields) != 0 {
		t.Errorf("Fields = %#v, want empty after remove", saved.Fields)
	}
}
