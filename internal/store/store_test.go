package store_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEntry(kind, name, body string) entry.Entry {
	return entry.Entry{
		Kind: kind,
		Name: name,
		Body: body,
	}
}

// TestStore_CRUD exercises create, get, update, delete.
func TestStore_CRUD(t *testing.T) {
	s := openTestStore(t)

	e, err := s.Create(makeEntry("note", "first note", "hello world"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.Status != entry.StatusActive {
		t.Errorf("expected active, got %s", e.Status)
	}

	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "first note" {
		t.Errorf("Name: got %q, want %q", got.Name, "first note")
	}

	got.Name = "updated note"
	got.Tags = []string{"important"}
	got, err = s.Update(got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Name != "updated note" {
		t.Errorf("updated Name: got %q", got.Name)
	}

	byName, err := s.GetByName("updated note")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if byName.ID != e.ID {
		t.Errorf("GetByName ID mismatch")
	}

	if err := s.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got.Status != entry.StatusArchived {
		t.Errorf("expected archived after delete, got %s", got.Status)
	}
	all, err := s.List("", nil)
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 active entries after delete, got %d", len(all))
	}
}

// TestStore_Delivery verifies the delivery mode defaults to init, persists,
// and can be toggled via SetDelivery.
func TestStore_Delivery(t *testing.T) {
	s := openTestStore(t)

	e, err := s.Create(makeEntry("rule", "use jj", "Use jj instead of git."))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.Delivery != entry.DeliveryInit {
		t.Errorf("expected default delivery %q, got %q", entry.DeliveryInit, e.Delivery)
	}

	if err := s.SetDelivery(e.ID, entry.DeliveryOnDemand); err != nil {
		t.Fatalf("SetDelivery: %v", err)
	}

	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Delivery != entry.DeliveryOnDemand {
		t.Errorf("expected delivery %q after SetDelivery, got %q", entry.DeliveryOnDemand, got.Delivery)
	}

	// Update must preserve delivery rather than reset it.
	got.Body = "Use jujutsu (jj) instead of git."
	if _, err := s.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if reloaded.Delivery != entry.DeliveryOnDemand {
		t.Errorf("Update reset delivery: got %q, want %q", reloaded.Delivery, entry.DeliveryOnDemand)
	}

	if err := s.SetDelivery("nonexistent", entry.DeliveryInit); err == nil {
		t.Error("expected error for SetDelivery on missing id, got nil")
	}
}

// TestStore_ListFiltersKind verifies kind filtering in List.
func TestStore_ListFiltersKind(t *testing.T) {
	s := openTestStore(t)

	_, _ = s.Create(makeEntry("note", "note-1", "body"))
	_, _ = s.Create(makeEntry("note", "note-2", "body"))
	_, _ = s.Create(makeEntry("task", "task-1", "body"))

	notes, err := s.List("note", nil)
	if err != nil {
		t.Fatalf("List note: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}

	tasks, err := s.List("task", nil)
	if err != nil {
		t.Fatalf("List task: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	all, err := s.List("", nil)
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 total, got %d", len(all))
	}
}

// TestStore_ListFiltersTags verifies AND tag filtering.
func TestStore_ListFiltersTags(t *testing.T) {
	s := openTestStore(t)

	e1 := entry.Entry{Kind: "note", Name: "tagged-ab", Tags: []string{"a", "b"}}
	e2 := entry.Entry{Kind: "note", Name: "tagged-a", Tags: []string{"a"}}
	e3 := entry.Entry{Kind: "note", Name: "tagged-c", Tags: []string{"c"}}
	_, _ = s.Create(e1)
	_, _ = s.Create(e2)
	_, _ = s.Create(e3)

	res, err := s.List("", []string{"a"})
	if err != nil {
		t.Fatalf("List tags [a]: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("expected 2 entries with tag 'a', got %d", len(res))
	}

	res, err = s.List("", []string{"a", "b"})
	if err != nil {
		t.Fatalf("List tags [a,b]: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 entry with tags 'a' and 'b', got %d", len(res))
	}

	res, err = s.List("", []string{"z"})
	if err != nil {
		t.Fatalf("List tags [z]: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 entries with tag 'z', got %d", len(res))
	}
}

// TestStore_SearchFTS verifies full-text search finds matching entries.
func TestStore_SearchFTS(t *testing.T) {
	s := openTestStore(t)

	_, _ = s.Create(makeEntry("arch", "service design", "we run on a monolith architecture"))
	_, _ = s.Create(makeEntry("arch", "microservices", "split into many services"))

	results, err := s.Search("monolith")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "service design" {
		t.Errorf("unexpected result: %s", results[0].Name)
	}
}

// TestStore_ActiveOnlyFiltering ensures pending entries are excluded from List and Search.
func TestStore_ActiveOnlyFiltering(t *testing.T) {
	s := openTestStore(t)

	_, _ = s.Create(makeEntry("note", "active-one", "visible content"))

	proposal := entry.Entry{Kind: "note", Name: "pending-one", Body: "hidden pending content"}
	_, _ = s.CreatePending(proposal)

	all, err := s.List("", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("List: expected 1 active entry, got %d", len(all))
	}

	results, err := s.Search("hidden")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search: expected 0 results (pending excluded), got %d", len(results))
	}
}

// TestStore_CreatePendingAndApproveNew checks that approving a proposal with no target creates a new active entry.
func TestStore_CreatePendingAndApproveNew(t *testing.T) {
	s := openTestStore(t)

	proposal := entry.Entry{
		Kind:       "note",
		Name:       "agent idea",
		Body:       "some great idea",
		ProposedBy: "agent-1",
	}
	p, err := s.CreatePending(proposal)
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	if p.Status != entry.StatusPending {
		t.Errorf("expected pending, got %s", p.Status)
	}
	if p.ProposedAt == nil {
		t.Error("expected ProposedAt to be set")
	}

	approved, err := s.Approve(p.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != entry.StatusActive {
		t.Errorf("expected active after approval, got %s", approved.Status)
	}
	if approved.ProposedAt != nil {
		t.Error("expected ProposedAt to be cleared after approval")
	}

	// The proposal row should now be active (same ID, promoted in place).
	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get after approve: %v", err)
	}
	if got.Status != entry.StatusActive {
		t.Errorf("DB status: expected active, got %s", got.Status)
	}

	// Should appear in active list now.
	all, err := s.List("", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 active entry, got %d", len(all))
	}
}

// TestStore_CreatePendingAndApproveUpdate checks merge into an existing target.
func TestStore_CreatePendingAndApproveUpdate(t *testing.T) {
	s := openTestStore(t)

	original, err := s.Create(entry.Entry{
		Kind:   "note",
		Name:   "original name",
		Body:   "original body",
		Fields: map[string]string{"priority": "low"},
	})
	if err != nil {
		t.Fatalf("Create original: %v", err)
	}

	// Propose an update: change body and add a field, leave name empty.
	proposal := entry.Entry{
		Kind:       "note",
		Name:       "",
		Body:       "updated body",
		Fields:     map[string]string{"priority": "high", "owner": "alice"},
		TargetID:   original.ID,
		ProposedBy: "agent-2",
	}
	p, err := s.CreatePending(proposal)
	if err != nil {
		t.Fatalf("CreatePending update: %v", err)
	}

	_, err = s.Approve(p.ID)
	if err != nil {
		t.Fatalf("Approve update: %v", err)
	}

	// Proposal row should be gone.
	if _, err := s.Get(p.ID); err == nil {
		t.Error("expected proposal to be deleted after approve-update")
	}

	// Target should have merged fields.
	target, err := s.Get(original.ID)
	if err != nil {
		t.Fatalf("Get target after approve: %v", err)
	}
	if target.Name != "original name" {
		t.Errorf("Name should be unchanged: got %q", target.Name)
	}
	if target.Body != "updated body" {
		t.Errorf("Body should be updated: got %q", target.Body)
	}
	if target.Fields["priority"] != "high" {
		t.Errorf("priority field: got %q, want high", target.Fields["priority"])
	}
	if target.Fields["owner"] != "alice" {
		t.Errorf("owner field: got %q, want alice", target.Fields["owner"])
	}
}

// TestStore_ApproveAll promotes all pending entries at once.
func TestStore_ApproveAll(t *testing.T) {
	s := openTestStore(t)

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("pending-%d", i)
		_, _ = s.CreatePending(entry.Entry{Kind: "note", Name: name, Body: "body"})
	}

	count, err := s.PendingCount()
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 pending, got %d", count)
	}

	approved, err := s.ApproveAll()
	if err != nil {
		t.Fatalf("ApproveAll: %v", err)
	}
	if len(approved) != 3 {
		t.Errorf("ApproveAll returned %d entries, want 3", len(approved))
	}

	count, _ = s.PendingCount()
	if count != 0 {
		t.Errorf("expected 0 pending after ApproveAll, got %d", count)
	}

	all, _ := s.List("", nil)
	if len(all) != 3 {
		t.Errorf("expected 3 active entries after ApproveAll, got %d", len(all))
	}
}

// TestStore_Reject verifies proposal is deleted and count decrements.
func TestStore_Reject(t *testing.T) {
	s := openTestStore(t)

	p1, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "p1", Body: "body"})
	p2, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "p2", Body: "body"})

	count, _ := s.PendingCount()
	if count != 2 {
		t.Errorf("expected 2 pending, got %d", count)
	}

	if err := s.Reject(p1.ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	count, _ = s.PendingCount()
	if count != 1 {
		t.Errorf("expected 1 pending after reject, got %d", count)
	}

	if _, err := s.Get(p1.ID); err == nil {
		t.Error("expected p1 to be deleted")
	}
	if _, err := s.Get(p2.ID); err != nil {
		t.Errorf("p2 should still exist: %v", err)
	}

	// Reject non-pending should error.
	active, _ := s.Create(makeEntry("note", "active", "body"))
	if err := s.Reject(active.ID); err == nil {
		t.Error("expected error rejecting an active entry")
	}
}

// TestStore_PendingCount verifies accurate count through creates and approvals.
func TestStore_PendingCount(t *testing.T) {
	s := openTestStore(t)

	c, _ := s.PendingCount()
	if c != 0 {
		t.Errorf("initial count: want 0, got %d", c)
	}

	p1, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "n1"})
	p2, _ := s.CreatePending(entry.Entry{Kind: "note", Name: "n2"})
	_, _ = s.CreatePending(entry.Entry{Kind: "note", Name: "n3"})

	c, _ = s.PendingCount()
	if c != 3 {
		t.Errorf("after 3 creates: want 3, got %d", c)
	}

	_, _ = s.Approve(p1.ID)
	c, _ = s.PendingCount()
	if c != 2 {
		t.Errorf("after 1 approve: want 2, got %d", c)
	}

	_ = s.Reject(p2.ID)
	c, _ = s.PendingCount()
	if c != 1 {
		t.Errorf("after 1 reject: want 1, got %d", c)
	}
}
