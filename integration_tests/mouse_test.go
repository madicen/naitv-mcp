package integration_tests

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_MouseEntryRowClick simulates a mouse click event on the entries
// tab by delivering a tea.MouseClickMsg. Verifies the model handles it without panic.
func TestJourney_MouseEntryRowClick(t *testing.T) {
	st := newTestDB(t)

	_, err := st.Create(entry.Entry{Kind: "repo", Name: "clickable-repo", Body: "click me"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	m := newTestModel(t, st)
	loaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, loaded)

	// Simulate a left-click mouse event in the entries area.
	mouseMsg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      20,
		Y:      5,
	}
	m = updateModel(m, mouseMsg)

	// Should not panic; view should still render.
	view := m.View().Content
	if view == "" {
		t.Error("expected non-empty view after mouse click")
	}
}

// TestJourney_MouseReviewActions simulates clicking in the review tab area
// and verifies the model handles the event without panic.
func TestJourney_MouseReviewActions(t *testing.T) {
	st := newTestDB(t)

	prop, err := st.CreatePending(entry.Entry{
		Kind:       "note",
		Name:       "mouse-proposal",
		Body:       "click to approve",
		ProposedBy: "claude",
	})
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	m := newTestModel(t, st)

	// Load proposals.
	loaded := review.LoadProposalsCmd(st)()
	m = updateModel(m, loaded)

	// Simulate a click on the "tab:review" area (approximate coords).
	mouseMsg := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      30,
		Y:      0,
	}
	m = updateModel(m, mouseMsg)

	// Simulate a click in the content area (approximate approve button region).
	mouseMsg2 := tea.MouseClickMsg{
		Button: tea.MouseLeft,
		X:      60,
		Y:      15,
	}
	m = updateModel(m, mouseMsg2)

	// Verify model still renderable.
	view := m.View().Content
	if view == "" {
		t.Error("expected non-empty view after mouse interactions")
	}

	// Clean up pending entry from store if not approved.
	_ = st.Reject(prop.ID)
}
