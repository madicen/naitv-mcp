package setup_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/madicen/naitv-mcp/internal/setup"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestSetProjectUpdatesWorkingDir(t *testing.T) {
	st := openStore(t)
	if _, err := st.Create(entry.Entry{
		Kind: "tool",
		Name: "lint",
		Body: "Lint project",
		Fields: map[string]string{
			"exec":        "golangci-lint run",
			"working_dir": "/old",
		},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	result, err := setup.SetProject(st, "/new/project", false, false)
	if err != nil {
		t.Fatalf("SetProject: %v", err)
	}
	if len(result.Updated) != 1 || result.Updated[0] != "lint" {
		t.Fatalf("Updated = %#v", result.Updated)
	}
	got, err := st.GetByName("lint")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Fields["working_dir"] != "/new/project" {
		t.Fatalf("working_dir = %q", got.Fields["working_dir"])
	}
}

func TestSetProjectSkippedAndLintToggle(t *testing.T) {
	st := openStore(t)
	if _, err := st.Create(entry.Entry{
		Kind: "tool", Name: "build", Body: "build",
		Fields: map[string]string{"exec": "true", "working_dir": "/proj"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create(entry.Entry{
		Kind: "tool", Name: "lint", Body: "lint",
		Fields: map[string]string{"exec": "true", "working_dir": "/proj", "disabled": "true"},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := setup.SetProject(st, "/proj", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "build" {
		t.Fatalf("Skipped = %#v", result.Skipped)
	}
	if len(result.Updated) != 1 || result.Updated[0] != "lint" {
		t.Fatalf("Updated = %#v", result.Updated)
	}
}

func TestContinueConfigIncludesTools(t *testing.T) {
	text := setup.ContinueConfig([]string{"lint", "test"}, "/usr/bin/naitv-mcp")
	for _, want := range []string{"lint", "test", "/usr/bin/naitv-mcp", "mcpServers"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in config", want)
		}
	}
	empty := setup.ContinueConfig(nil, "/bin/naitv-mcp")
	if !strings.Contains(empty, "no executable tools registered yet") {
		t.Fatalf("empty config = %q", empty)
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

func TestResolveDir(t *testing.T) {
	dir := t.TempDir()
	got, err := setup.ResolveDir(dir)
	if err != nil {
		t.Fatalf("ResolveDir: %v", err)
	}
	if got == "" {
		t.Fatal("expected absolute path")
	}
	if _, err := setup.ResolveDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error for missing path")
	}
	if got, err := setup.ResolveDir(""); err != nil || got != "" {
		t.Fatalf("empty = %q, %v", got, err)
	}
}
