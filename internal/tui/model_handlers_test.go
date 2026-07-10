package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/plugins"
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

func TestHandleReviewRequest_RejectAndApproveAll(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)
	p1, _ := st.CreatePending(entry.Entry{Kind: "note", Name: "one", Body: "x"})
	p2, _ := st.CreatePending(entry.Entry{Kind: "note", Name: "two", Body: "y"})
	runPluginCmd(t, m, review.LoadProposalsCmd(st))
	runPluginCmd(t, m, m.handleReviewRequest(&review.Request{RejectSelected: true}))
	if _, err := st.Get(p1.ID); err == nil {
		t.Fatal("expected reject to remove proposal")
	}
	runPluginCmd(t, m, review.LoadProposalsCmd(st))
	runPluginCmd(t, m, m.handleReviewRequest(&review.Request{ApproveAll: true}))
	if _, err := st.GetByName("two"); err != nil {
		t.Fatalf("approve all failed: %v", err)
	}
	_ = p2
}

func TestHandlePluginsRequest_FetchRegistry(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	regPath := filepath.Join(t.TempDir(), "registry.json")
	_ = os.WriteFile(regPath, []byte(`{"plugins":[{"name":"demo","version":"1.0.0","url":"file:///x","description":"d"}]}`), 0o644)
	runPluginCmd(t, m, m.handlePluginsRequest(&plugins.Request{FetchRegistry: true, Source: regPath}))
}
