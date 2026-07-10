package integration_tests

import (
	"testing"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestJourney_MouseEntryRowClick(t *testing.T) {
	st := newTestDB(t)
	_, err := st.Create(entry.Entry{Kind: "repo", Name: "clickable-repo", Body: "click me"})
	if err != nil { t.Fatalf("create: %v", err) }
	m := newTestModel(t, st)
	m = updateModel(m, entries.LoadEntriesCmd(st, "")())
	m = clickZone(t, m, zones.EntriesRow(0))
	if m.View().Content == "" { t.Fatal("expected view") }
}

func TestJourney_MouseTabSwitch(t *testing.T) {
	m := clickZone(t, newTestModel(t, newTestDB(t)), zones.TabReview)
	if m.View().Content == "" { t.Fatal("expected view") }
}

func TestJourney_MouseReviewApprove(t *testing.T) {
	st := newTestDB(t)
	prop, err := st.CreatePending(entry.Entry{Kind: "note", Name: "mouse-proposal", Body: "x", ProposedBy: "claude"})
	if err != nil { t.Fatalf("create pending: %v", err) }
	m := newTestModel(t, st)
	m = updateModel(m, review.LoadProposalsCmd(st)())
	m = clickZone(t, m, zones.TabReview)
	clickZone(t, m, zones.ReviewApprove)
	for _, p := range must(st.ListPending()) {
		if p.ID == prop.ID { t.Fatal("expected approved") }
	}
}

func must[T any](v T, err error) T { if err != nil { panic(err) }; return v }
