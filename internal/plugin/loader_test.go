package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_LocalManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.json")
	manifest := `{
		"name": "demo",
		"version": "1.0.0",
		"entries": [{"kind": "rule", "name": "demo-rule", "body": "hello"}]
	}`
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Name != "demo" || len(m.Entries) != 1 {
		t.Fatalf("manifest = %#v", m)
	}
}

func TestRegistry_Find(t *testing.T) {
	reg := Registry{Plugins: []RegistryEntry{{Name: "alpha", Version: "1"}}}
	if reg.Find("alpha") == nil || reg.Find("missing") != nil {
		t.Fatal("Find mismatch")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadRegistry_LocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	data := `{"plugins":[{"name":"demo","version":"1.0.0","url":"file:///x","description":"d"}]}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(reg.Plugins) != 1 || reg.Plugins[0].Name != "demo" {
		t.Fatalf("registry = %#v", reg)
	}
}
