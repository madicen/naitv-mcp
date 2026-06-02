package integration_tests

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// TestJourney_AgentProposeNew creates a pending proposal, switches to the
// review tab via message injection, approves it, and verifies it appears
// in the active list.
func TestJourney_AgentProposeNew(t *testing.T) {
	st := newTestDB(t)

	prop, err := st.CreatePending(entry.Entry{
		Kind:       "repo",
		Name:       "agent-proposed-repo",
		Body:       "Proposed by agent.",
		ProposedBy: "claude",
	})
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	m := newTestModel(t, st)

	// Deliver proposals loaded message so the model knows about proposals.
	loaded := review.LoadProposalsCmd(st)()
	m = updateModel(m, loaded)

	// Deliver approval message directly (simulates the store command result).
	_, err = st.Approve(prop.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	approveMsg := review.ProposalApprovedMsg{Entry: prop}
	m = updateModel(m, approveMsg)

	active, err := st.List("", nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, e := range active {
		if e.Name == "agent-proposed-repo" {
			found = true
		}
	}
	if !found {
		t.Error("expected agent-proposed-repo in active entries after approval")
	}

	// Reload entries in model and check view.
	entriesLoaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, entriesLoaded)
	view := m.View()
	if !strings.Contains(view, "agent-proposed-repo") {
		t.Errorf("expected agent-proposed-repo in view, got:\n%s", view)
	}
}

// TestJourney_AgentProposeUpdate creates an active entry, creates a pending
// update proposal targeting it, approves it, and verifies the update was applied.
func TestJourney_AgentProposeUpdate(t *testing.T) {
	st := newTestDB(t)

	orig, err := st.Create(entry.Entry{Kind: "note", Name: "update-target", Body: "old body"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	prop, err := st.CreatePending(entry.Entry{
		Kind:       "note",
		Name:       "update-target",
		Body:       "new body from agent",
		TargetID:   orig.ID,
		ProposedBy: "claude",
	})
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	// Approve in store directly.
	_, err = st.Approve(prop.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}

	updated, err := st.Get(orig.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Body != "new body from agent" {
		t.Errorf("expected updated body, got %q", updated.Body)
	}

	// Deliver the message flow through the model too.
	m := newTestModel(t, st)
	approveMsg := review.ProposalApprovedMsg{Entry: updated}
	m = updateModel(m, approveMsg)
	_ = m.View()
}

// TestJourney_EditBeforeApprove selects a proposal in review, simulates edit
// by updating the store directly, then approves. Verifies final state.
func TestJourney_EditBeforeApprove(t *testing.T) {
	st := newTestDB(t)

	prop, err := st.CreatePending(entry.Entry{
		Kind:       "workflow",
		Name:       "edit-before-approve",
		Body:       "original proposal body",
		ProposedBy: "claude",
	})
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	m := newTestModel(t, st)

	// Deliver proposals loaded msg.
	loaded := review.LoadProposalsCmd(st)()
	m = updateModel(m, loaded)

	// Simulate editing the pending proposal before approve.
	prop.Body = "edited before approve"
	_, err = st.Update(prop)
	if err != nil {
		t.Fatalf("update proposal: %v", err)
	}

	// Approve in store.
	approved, err := st.Approve(prop.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}

	if approved.Body != "edited before approve" {
		t.Errorf("expected edited body in approved entry, got %q", approved.Body)
	}

	// Deliver approval through model.
	approveMsg := review.ProposalApprovedMsg{Entry: approved}
	m = updateModel(m, approveMsg)

	// Verify active list.
	active, _ := st.List("", nil)
	found := false
	for _, e := range active {
		if e.Name == "edit-before-approve" && strings.Contains(e.Body, "edited") {
			found = true
		}
	}
	if !found {
		t.Error("expected edit-before-approve entry in active list")
	}

	_ = m.View()
}

// TestJourney_RejectProposal creates a pending proposal, rejects it, and
// verifies it is gone from pending.
func TestJourney_RejectProposal(t *testing.T) {
	st := newTestDB(t)

	prop, err := st.CreatePending(entry.Entry{
		Kind:       "fact",
		Name:       "reject-me",
		Body:       "this will be rejected",
		ProposedBy: "claude",
	})
	if err != nil {
		t.Fatalf("create pending: %v", err)
	}

	m := newTestModel(t, st)

	// Load proposals.
	loaded := review.LoadProposalsCmd(st)()
	m = updateModel(m, loaded)

	// Reject in store.
	err = st.Reject(prop.ID)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}

	// Deliver rejection message through model.
	rejectMsg := review.ProposalRejectedMsg{ID: prop.ID}
	m = updateModel(m, rejectMsg)

	pending, err := st.ListPending()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	for _, p := range pending {
		if p.ID == prop.ID {
			t.Error("expected proposal to be gone from pending after rejection")
		}
	}

	_ = m.View()
}

// TestJourney_ApproveAll creates 3 pending proposals, approves all via store,
// and verifies 0 pending remain.
func TestJourney_ApproveAll(t *testing.T) {
	st := newTestDB(t)

	for i := 0; i < 3; i++ {
		name := "proposal-" + string(rune('A'+i))
		_, err := st.CreatePending(entry.Entry{
			Kind:       "note",
			Name:       name,
			Body:       "pending body",
			ProposedBy: "claude",
		})
		if err != nil {
			t.Fatalf("create pending %s: %v", name, err)
		}
	}

	m := newTestModel(t, st)

	// Load proposals into model.
	loaded := review.LoadProposalsCmd(st)()
	m = updateModel(m, loaded)

	// ApproveAll in store.
	approved, err := st.ApproveAll()
	if err != nil {
		t.Fatalf("approve all: %v", err)
	}
	if len(approved) != 3 {
		t.Errorf("expected 3 approved, got %d", len(approved))
	}

	// Deliver AllApprovedMsg through model.
	allApprovedMsg := review.AllApprovedMsg{Entries: approved}
	m = updateModel(m, allApprovedMsg)

	pending, err := st.ListPending()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after approve all, got %d", len(pending))
	}

	// Reload entries and check view.
	entriesLoaded := entries.LoadEntriesCmd(st, "")()
	m = updateModel(m, entriesLoaded)
	_ = m.View()
}
