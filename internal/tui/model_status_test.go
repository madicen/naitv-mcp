package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestPendingCountBadge(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)
	if _, err := st.CreatePending(entry.Entry{Kind: "note", Name: "badge-prop", Body: "x", ProposedBy: "test"}); err != nil {
		t.Fatalf("create pending: %v", err)
	}
	runPluginCmd(t, m, m.loadPendingCount())
	if !containsSubstring(m.View().Content, "(1)") {
		t.Error("expected pending badge")
	}
	m.activeTab = tabReview
	plain := stripANSI(m.renderTabBar())
	if !strings.Contains(plain, "Review (1)") {
		t.Fatalf("expected badge tight against Review, got %q", plain)
	}
	if strings.Contains(plain, "38;5") {
		t.Fatalf("ANSI leaked into tab bar text: %q", plain)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestStatusExpiry(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(80, 24)
	m.setStatus("hello")
	if !containsSubstring(m.View().Content, "hello") {
		t.Error("expected fresh status")
	}
	m.statusExpiry = time.Now().Add(-time.Second)
	if containsSubstring(m.View().Content, "hello") {
		t.Error("expected expired status hidden")
	}
}

func TestHandleEntriesRequest_CopyBody(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)
	if _, err := st.Create(entry.Entry{Kind: "note", Name: "copy-src", Body: "payload"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	loaded := entries.LoadEntriesCmd(st, "")()
	next, _ := m.Update(loaded)
	m = next.(*Model)
	runPluginCmd(t, m, m.handleEntriesRequest(&entries.Request{CopyBody: true}))
	if !containsSubstring(m.View().Content, "Copied body") {
		t.Error("expected copy status")
	}
}

func containsSubstring(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
