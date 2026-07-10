package layout_test

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/layout"
)

func TestSplitWidths(t *testing.T) {
	listW, detailW := layout.SplitWidths(100)
	if listW != 35 || detailW != 64 {
		t.Fatalf("got %d, %d", listW, detailW)
	}
}

func TestContentHeightFloorsAtOne(t *testing.T) {
	if got := layout.ContentHeight(1, 5); got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestViewportSize(t *testing.T) {
	w, h := layout.ViewportSize(10, 8)
	if w != 8 || h != 6 {
		t.Fatalf("got %d, %d", w, h)
	}
}

func TestTruncate(t *testing.T) {
	if layout.Truncate("hello", 0) != "" {
		t.Fatal("expected empty")
	}
	if layout.Truncate("hello", 10) != "hello" {
		t.Fatal("expected unchanged")
	}
	got := layout.Truncate("hello world", 8)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis: %q", got)
	}
}
