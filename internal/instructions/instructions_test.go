package instructions

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestRenderEmpty(t *testing.T) {
	got := Render(nil)
	if !strings.Contains(got, "No context entries yet") {
		t.Fatalf("expected empty-state message, got:\n%s", got)
	}
}

func TestRenderOrdersKnownKindsAndIncludesContent(t *testing.T) {
	entries := []entry.Entry{
		{Kind: "repo", Name: "jj-tui", Body: "Reference for TUI patterns.", Tags: []string{"go"}, Fields: map[string]string{"path": "~/dev/jj-tui"}},
		{Kind: "rule", Name: "use jj", Body: "Use jj instead of git."},
		{Kind: "workflow", Name: "commits", Body: "Do work in commits; let the user check them in."},
	}

	got := Render(entries)

	// Rules section must come before Repositories section.
	rulesIdx := strings.Index(got, "## Rules")
	reposIdx := strings.Index(got, "## Repositories")
	if rulesIdx == -1 || reposIdx == -1 {
		t.Fatalf("missing expected sections, got:\n%s", got)
	}
	if rulesIdx > reposIdx {
		t.Fatalf("expected Rules before Repositories, got:\n%s", got)
	}

	for _, want := range []string{
		"Use jj instead of git.",
		"Do work in commits; let the user check them in.",
		"### jj-tui",
		"**path**: ~/dev/jj-tui",
		"_tags: go_",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestFilterInit(t *testing.T) {
	entries := []entry.Entry{
		{Kind: "rule", Name: "default", Body: "no delivery set"},
		{Kind: "rule", Name: "explicit-init", Body: "x", Delivery: entry.DeliveryInit},
		{Kind: "fact", Name: "secret", Body: "y", Delivery: entry.DeliveryOnDemand},
	}

	got := FilterInit(entries)
	if len(got) != 2 {
		t.Fatalf("expected 2 init entries, got %d", len(got))
	}
	for _, e := range got {
		if e.Name == "secret" {
			t.Errorf("on-demand entry should be excluded from init bundle")
		}
	}
}

func TestFilterInitByKinds(t *testing.T) {
	entries := []entry.Entry{
		{Kind: "rule", Name: "r1", Body: "a"},
		{Kind: "note", Name: "n1", Body: "b"},
		{Kind: "tool", Name: "t1", Body: "c"},
	}
	got := FilterInitByKinds(entries, []string{"rule", "tool"})
	if len(got) != 2 {
		t.Fatalf("got %d entries", len(got))
	}
	all := FilterInitByKinds(entries, nil)
	if len(all) != 3 {
		t.Fatalf("nil kinds should include all init kinds, got %d", len(all))
	}
}

func TestRenderUnknownKindAppended(t *testing.T) {
	entries := []entry.Entry{
		{Kind: "custom", Name: "thing", Body: "some body"},
	}
	got := Render(entries)
	if !strings.Contains(got, "## Custom") {
		t.Fatalf("expected unknown kind rendered with title-cased heading, got:\n%s", got)
	}
}
