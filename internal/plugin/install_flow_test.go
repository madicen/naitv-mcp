package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/madicen/naitv-mcp/internal/store"
)

func TestInstall_LocalPlugin(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.json")
	manifest := `{
		"name": "local-demo",
		"version": "0.1.0",
		"description": "Local test plugin",
		"entries": [
			{"kind": "rule", "name": "local-rule", "body": "hello"}
		]
	}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	result, err := Install(st, manifestPath)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if result.Manifest.Name != "local-demo" || len(result.Proposed) != 1 {
		t.Fatalf("result = %#v", result)
	}
	plugins, err := st.List("plugin", nil)
	if err != nil || len(plugins) != 1 || plugins[0].Name != "local-demo" {
		t.Fatalf("plugins = %#v, %v", plugins, err)
	}
}

func TestUninstall_LocalPlugin(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "plugin.json")
	manifest := `{
		"name": "rm-demo",
		"version": "0.1.0",
		"entries": [{"kind": "rule", "name": "rm-rule", "body": "x"}]
	}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if _, err := Install(st, manifestPath); err != nil {
		t.Fatalf("Install: %v", err)
	}
	pending, _ := st.ListPending()
	for _, p := range pending {
		if _, err := st.Approve(p.ID); err != nil {
			t.Fatalf("Approve: %v", err)
		}
	}
	result, err := Uninstall(st, "rm-demo")
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if result.Name != "rm-demo" {
		t.Fatalf("result = %#v", result)
	}
}
