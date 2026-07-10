package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/plugins"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestHandlePluginsRequest_InstallFromManifest(t *testing.T) {
	st := openTestStore(t)
	m := New(st)
	m.SetDimensions(120, 40)

	manifestPath := writeTestManifest(t, plugin.Manifest{
		Name:    "test-plugin",
		Version: "1.0.0",
		Entries: []plugin.EntrySpec{
			{Kind: "rule", Name: "test-rule", Body: "from plugin"},
		},
	})

	cmd := m.handlePluginsRequest(&plugins.Request{Install: true, Source: manifestPath})
	runPluginCmd(t, m, cmd)

	installed, err := st.List("plugin", nil)
	if err != nil {
		t.Fatalf("list plugins: %v", err)
	}
	if len(installed) != 1 || installed[0].Name != "test-plugin" {
		t.Fatalf("installed plugins = %#v, want test-plugin", installed)
	}

	pending, err := st.ListPending()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 || pending[0].Name != "test-rule" {
		t.Fatalf("pending = %#v, want test-rule proposal", pending)
	}
}

func TestHandlePluginsRequest_Uninstall(t *testing.T) {
	st := openTestStore(t)
	m := New(st)

	manifestPath := writeTestManifest(t, plugin.Manifest{
		Name:    "uninstall-me",
		Version: "1.0.0",
		Entries: []plugin.EntrySpec{
			{Kind: "note", Name: "plugin-entry", Body: "bye"},
		},
	})

	installCmd := m.handlePluginsRequest(&plugins.Request{Install: true, Source: manifestPath})
	runPluginCmd(t, m, installCmd)

	pending, _ := st.ListPending()
	for _, p := range pending {
		if _, err := st.Approve(p.ID); err != nil {
			t.Fatalf("approve %s: %v", p.Name, err)
		}
	}

	uninstallCmd := m.handlePluginsRequest(&plugins.Request{Uninstall: true, Name: "uninstall-me"})
	runPluginCmd(t, m, uninstallCmd)

	installed, _ := st.List("plugin", nil)
	for _, e := range installed {
		if e.Name == "uninstall-me" {
			t.Fatal("expected plugin tracker removed after uninstall")
		}
	}
	active, _ := st.List("", nil)
	for _, e := range active {
		if e.Name == "plugin-entry" {
			t.Fatal("expected plugin entries removed after uninstall")
		}
	}
}

func TestHandlePluginsRequest_RefreshPendingCount(t *testing.T) {
	st := openTestStore(t)
	m := New(st)

	if _, err := st.CreatePending(entry.Entry{
		Kind: "note", Name: "pending-one", Body: "x", ProposedBy: "test",
	}); err != nil {
		t.Fatalf("create pending: %v", err)
	}

	cmd := m.handlePluginsRequest(&plugins.Request{RefreshPendingCount: true})
	runPluginCmd(t, m, cmd)

	if m.pendingCount != 1 {
		t.Errorf("pendingCount = %d, want 1", m.pendingCount)
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func runPluginCmd(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	next, follow := m.Update(msg)
	m = next.(*Model)
	for follow != nil {
		msg = follow()
		if msg == nil {
			break
		}
		next, follow = m.Update(msg)
		m = next.(*Model)
	}
}

func writeTestManifest(t *testing.T, manifest plugin.Manifest) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), manifest.Name+".json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}
