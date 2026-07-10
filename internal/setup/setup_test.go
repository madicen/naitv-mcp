package setup_test

import (
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

func TestContinueConfigIncludesTools(t *testing.T) {
	text := setup.ContinueConfig([]string{"lint", "test"}, "/usr/bin/naitv-mcp")
	for _, want := range []string{"lint", "test", "/usr/bin/naitv-mcp", "mcpServers"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in config", want)
		}
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
