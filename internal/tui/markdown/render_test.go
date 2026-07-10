package markdown_test

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/tui/markdown"
)

func TestRenderBodyCaches(t *testing.T) {
	r := markdown.NewRenderer(40)
	first := r.RenderBody("id1", "# Title\n\nParagraph.")
	second := r.RenderBody("id1", "# Title\n\nParagraph.")
	if first == "" {
		t.Fatal("expected rendered output")
	}
	if first != second {
		t.Fatal("expected cache hit")
	}
}

func TestRenderBodyEmpty(t *testing.T) {
	r := markdown.NewRenderer(40)
	if r.RenderBody("id", "") != "" {
		t.Fatal("expected empty")
	}
}

func TestSetWidthClearsCache(t *testing.T) {
	r := markdown.NewRenderer(40)
	body := "**bold** text"
	a := r.RenderBody("id", body)
	r.SetWidth(60)
	b := r.RenderBody("id", body)
	if a == b && !strings.Contains(a, "bold") {
		t.Fatalf("unexpected render: %q / %q", a, b)
	}
}
