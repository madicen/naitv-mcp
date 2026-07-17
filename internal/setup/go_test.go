package setup_test

import (
	"path/filepath"
	"testing"

	"github.com/madicen/naitv-mcp/internal/setup"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestSetProject_PreservesProjectRootPlaceholder(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	params := `[{"name":"project_root","description":"root","required":true}]`
	if _, err := st.Create(entry.Entry{
		Kind: "tool", Name: "build", Body: "build",
		Fields: map[string]string{
			"exec":        "go build ./...",
			"working_dir": "/Users/me/Documents/GitHub/jj-tui",
			"params":      params,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Create(entry.Entry{
		Kind: "tool", Name: "legacy-tool", Body: "legacy",
		Fields: map[string]string{
			"exec":        "echo hi",
			"working_dir": "/old",
		},
	}); err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	result, err := setup.SetProject(st, projectDir, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated = %#v, want 2 tools", result.Updated)
	}

	build, err := st.GetByName("build")
	if err != nil {
		t.Fatal(err)
	}
	if build.Fields["working_dir"] != setup.ProjectRootPlaceholder {
		t.Fatalf("build working_dir = %q, want %q", build.Fields["working_dir"], setup.ProjectRootPlaceholder)
	}

	legacy, err := st.GetByName("legacy-tool")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Fields["working_dir"] != projectDir {
		t.Fatalf("legacy working_dir = %q, want %q", legacy.Fields["working_dir"], projectDir)
	}
}
