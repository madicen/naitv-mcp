package mcp

import (
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestFormatEntry(t *testing.T) {
	text := formatEntry(entry.Entry{
		Kind:   "rule",
		Name:   "use-jj",
		Body:   "line one\nline two",
		Tags:   []string{"vcs"},
		Fields: map[string]string{"key": "val"},
	})
	for _, want := range []string{"[rule] use-jj", "tags: vcs", "key: val", "line one", "line two"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestFormatEntriesEmpty(t *testing.T) {
	if got := formatEntries(nil); got != "No entries found." {
		t.Fatalf("got %q", got)
	}
}

func TestFormatToolDefs(t *testing.T) {
	text := formatToolDefs([]tools.Def{{
		Name:        "lint",
		Exec:        "golangci-lint run",
		Timeout:     0,
		Description: "Lint the project",
		Params:      []tools.Param{{Name: "path", Description: "Package path", Required: true}},
	}})
	for _, want := range []string{"tool: lint", "exec:", "params:", "{path}", "Lint the project"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestFormatRegistry(t *testing.T) {
	text := formatRegistry(plugin.Registry{Plugins: []plugin.RegistryEntry{{
		Name:        "demo",
		Version:     "1.0.0",
		Description: "Demo plugin",
		Tags:        []string{"go"},
	}}})
	for _, want := range []string{"demo", "1.0.0", "Demo plugin", "tags: go", "install_plugin"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestParseTagsAndFields(t *testing.T) {
	if got := parseTags("a, b,,c"); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("parseTags = %#v", got)
	}
	if parseTags("") != nil {
		t.Fatal("expected nil for empty tags")
	}
	fields, err := parseFields(`{"x":"y"}`)
	if err != nil || fields["x"] != "y" {
		t.Fatalf("parseFields = %#v, %v", fields, err)
	}
	if _, err := parseFields("not-json"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestProposeEntry(t *testing.T) {
	st := openStore(t)
	result, err := proposeEntry(st, entryProposalSpec{
		Kind: "note",
		Name: "demo",
		Body: "hello",
	})
	if err != nil {
		t.Fatalf("proposeEntry: %v", err)
	}
	if result.Status != "queued" || result.ProposalID == "" {
		t.Fatalf("result = %#v", result)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
