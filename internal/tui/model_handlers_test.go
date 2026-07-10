package tui

import (
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestHandleReviewRequest_ApproveSelected(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)
	pending, err := st.CreatePending(entry.Entry{Kind: "note", Name: "approve-me", Body: "x"})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	runPluginCmd(t, m, review.LoadProposalsCmd(st))
	runPluginCmd(t, m, m.handleReviewRequest(&review.Request{ApproveSelected: true}))
	got, err := st.GetByName("approve-me")
	if err != nil || got.Status != entry.StatusActive {
		t.Fatalf("after approve: %#v, %v", got, err)
	}
	_ = pending
}

func TestHandleEntriesRequest_UndoAfterUpdate(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)
	created, err := st.Create(entry.Entry{Kind: "note", Name: "undo-me", Body: "v1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	created.Body = "v2"
	if _, err := st.Update(created); err != nil {
		t.Fatalf("Update: %v", err)
	}
	records, err := st.History(created.ID)
	if err != nil || len(records) == 0 {
		t.Fatalf("History: %#v, %v", records, err)
	}
	runPluginCmd(t, m, entries.LoadEntriesCmd(st, ""))
	m.undoHistoryID = records[0].ID
	runPluginCmd(t, m, m.handleEntriesRequest(&entries.Request{Undo: true}))
	got, err := st.Get(created.ID)
	if err != nil || got.Body != "v1" {
		t.Fatalf("undo result: %#v, %v", got, err)
	}
}
