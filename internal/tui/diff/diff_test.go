package diff_test

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/diff"
)

func TestUnifiedShowsChanges(t *testing.T) {
	out := diff.Unified("body", "old line", "new line")
	if strings.Contains(out, "(no changes)") {
		t.Fatalf("expected diff output: %q", out)
	}
	if !strings.Contains(out, "+") || !strings.Contains(out, "-") {
		t.Fatalf("expected +/- hunks: %q", out)
	}
}

func TestUnifiedNoChanges(t *testing.T) {
	out := diff.Unified("", "same", "same")
	if !strings.Contains(out, "(no changes)") {
		t.Fatalf("got %q", out)
	}
}

func TestFieldsDiff(t *testing.T) {
	out := diff.FieldsDiff(
		map[string]string{"exec": "old"},
		map[string]string{"exec": "new"},
	)
	if out == "" || !strings.Contains(out, "field exec") {
		t.Fatalf("got %q", out)
	}
}
