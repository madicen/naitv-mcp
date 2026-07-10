package store_test

import (
	"testing"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestStore_HistoryAndRestoreVersion(t *testing.T) {
	s := openStore(t)
	created, err := s.Create(entry.Entry{Kind: "note", Name: "hist", Body: "v1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	created.Body = "v2"
	if _, err := s.Update(created); err != nil {
		t.Fatalf("Update: %v", err)
	}
	records, err := s.History(created.ID)
	if err != nil || len(records) == 0 {
		t.Fatalf("History = %#v, %v", records, err)
	}
	restored, err := s.RestoreVersion(records[0].ID)
	if err != nil {
		t.Fatalf("RestoreVersion: %v", err)
	}
	if restored.Body != "v1" {
		t.Fatalf("restored body = %q", restored.Body)
	}
}

func TestStore_PurgeAndReject(t *testing.T) {
	s := openStore(t)
	pending, err := s.CreatePending(entry.Entry{Kind: "note", Name: "reject-me", Body: "x"})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	if err := s.Reject(pending.ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	active, err := s.Create(entry.Entry{Kind: "note", Name: "purge-me", Body: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(active.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Purge(active.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, err := s.Get(active.ID); err == nil {
		t.Fatal("expected purged entry gone")
	}
}

func TestStore_SetDeliveryAndPendingCount(t *testing.T) {
	s := openStore(t)
	created, err := s.Create(entry.Entry{Kind: "note", Name: "delivery", Body: "x"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SetDelivery(created.ID, entry.DeliveryOnDemand); err != nil {
		t.Fatalf("SetDelivery: %v", err)
	}
	got, err := s.Get(created.ID)
	if err != nil || got.DeliveryOrDefault() != entry.DeliveryOnDemand {
		t.Fatalf("delivery = %#v, %v", got, err)
	}
	if _, err := s.CreatePending(entry.Entry{Kind: "note", Name: "pending", Body: "p"}); err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	count, err := s.PendingCount()
	if err != nil || count != 1 {
		t.Fatalf("PendingCount = %d, %v", count, err)
	}
}

func TestStore_OnChangeNotifies(t *testing.T) {
	s := openStore(t)
	called := 0
	s.OnChange(func() { called++ })
	if _, err := s.Create(entry.Entry{Kind: "note", Name: "chg", Body: "x"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if called != 1 {
		t.Fatalf("called = %d", called)
	}
}

func TestStore_ModTime(t *testing.T) {
	s := openStore(t)
	if s.DBPath() == "" {
		t.Fatal("expected db path")
	}
	if _, err := s.ModTime(); err != nil {
		t.Fatalf("ModTime: %v", err)
	}
}
