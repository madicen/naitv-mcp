package plugins

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	m := NewModel(zone.New())
	m.SetDimensions(120, 40)
	return m
}

func TestUpdate_FetchRegistry_SetsLoadingAndRequest(t *testing.T) {
	m := newTestModel(t)

	updated, req, cmd := m.Update(key("r"))

	if req == nil || !req.FetchRegistry {
		t.Fatalf("expected FetchRegistry request, got %#v", req)
	}
	if !updated.loading {
		t.Error("expected loading=true after registry fetch request")
	}
	if updated.status != "Fetching registry…" {
		t.Errorf("status = %q, want %q", updated.status, "Fetching registry…")
	}
	if cmd == nil {
		t.Error("expected spinner tick command while loading")
	}
}

func TestUpdate_RegistryLoaded_Success(t *testing.T) {
	m := newTestModel(t)
	m.loading = true

	reg := plugin.Registry{
		Plugins: []plugin.RegistryEntry{
			{Name: "demo-plugin", Description: "A demo", URL: "https://example.com/demo.json"},
		},
	}
	updated, req, _ := m.Update(RegistryLoadedMsg{Registry: reg})

	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if updated.loading {
		t.Error("expected loading=false after registry loaded")
	}
	if updated.mode != modeBrowse {
		t.Error("expected modeBrowse after successful registry load")
	}
	if len(updated.available) != 1 || updated.available[0].Name != "demo-plugin" {
		t.Errorf("available = %#v, want demo-plugin", updated.available)
	}
	if !strings.Contains(updated.status, "1 plugin") {
		t.Errorf("status = %q, want registry loaded message", updated.status)
	}
}

func TestUpdate_RegistryLoaded_Error(t *testing.T) {
	m := newTestModel(t)
	m.loading = true

	updated, req, _ := m.Update(RegistryLoadedMsg{Err: errors.New("network down")})

	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if updated.loading {
		t.Error("expected loading=false after registry error")
	}
	if !strings.Contains(updated.status, "Registry fetch failed") {
		t.Errorf("status = %q, want fetch failed message", updated.status)
	}
}

func TestUpdate_PluginsLoaded(t *testing.T) {
	m := newTestModel(t)

	entries := []entry.Entry{
		{Kind: "plugin", Name: "installed-one"},
		{Kind: "plugin", Name: "installed-two"},
	}
	updated, req, _ := m.Update(PluginsLoadedMsg{Entries: entries})

	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if len(updated.installed) != 2 {
		t.Fatalf("installed len = %d, want 2", len(updated.installed))
	}
	if !updated.installedNames["installed-one"] || !updated.installedNames["installed-two"] {
		t.Errorf("installedNames = %#v", updated.installedNames)
	}
}

func TestUpdate_CustomInstallInputMode(t *testing.T) {
	m := newTestModel(t)
	m.installed = []entry.Entry{{Kind: "plugin", Name: "p1"}}

	// Open custom install input.
	updated, req, _ := m.Update(key("i"))
	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if !updated.inputActive {
		t.Fatal("expected inputActive after pressing i in installed mode")
	}

	// Esc cancels input mode.
	updated, req, _ = updated.Update(keyType(tea.KeyEsc))
	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if updated.inputActive {
		t.Error("expected inputActive=false after esc")
	}

	// Re-open and reject empty source.
	updated, _, _ = updated.Update(key("i"))
	updated, req, _ = updated.Update(key("enter"))
	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if updated.inputActive {
		t.Error("expected input to stay open on empty source")
	}
	if !strings.Contains(updated.status, "Enter a plugin") {
		t.Errorf("status = %q, want empty-source prompt", updated.status)
	}

	// Type a source and confirm install.
	updated, _, _ = updated.Update(key("i"))
	updated.input.SetValue("./plugins/demo.json")
	updated, req, cmd = updated.Update(key("enter"))
	if req == nil || !req.Install || req.Source != "./plugins/demo.json" {
		t.Fatalf("expected install request for custom source, got %#v", req)
	}
	if !updated.loading {
		t.Error("expected loading=true after custom install submit")
	}
	if updated.inputActive {
		t.Error("expected inputActive=false after submit")
	}
	if cmd == nil {
		t.Error("expected spinner tick command while installing")
	}
}

func TestUpdate_BrowseInstall(t *testing.T) {
	m := newTestModel(t)
	m.mode = modeBrowse
	m.available = []plugin.RegistryEntry{
		{Name: "browse-plugin", URL: "https://example.com/browse.json"},
	}

	updated, req, cmd := m.Update(key("i"))

	if req == nil || !req.Install || req.Source != "browse-plugin" {
		t.Fatalf("expected browse install request, got %#v", req)
	}
	if !strings.Contains(updated.status, "Installing browse-plugin") {
		t.Errorf("status = %q", updated.status)
	}
	if !updated.loading || cmd == nil {
		t.Error("expected loading state with spinner cmd")
	}
}

func TestUpdate_BrowseInstallAlreadyInstalled(t *testing.T) {
	m := newTestModel(t)
	m.mode = modeBrowse
	m.available = []plugin.RegistryEntry{{Name: "existing-plugin"}}
	m.installedNames = map[string]bool{"existing-plugin": true}

	updated, req, _ := m.Update(key("i"))

	if req != nil {
		t.Fatalf("unexpected request %#v", req)
	}
	if !strings.Contains(updated.status, "already installed") {
		t.Errorf("status = %q, want already-installed message", updated.status)
	}
}

func TestUpdate_Uninstall(t *testing.T) {
	m := newTestModel(t)
	m.installed = []entry.Entry{{Kind: "plugin", Name: "remove-me"}}

	updated, req, cmd := m.Update(key("u"))

	if req == nil || !req.Uninstall || req.Name != "remove-me" {
		t.Fatalf("expected uninstall request, got %#v", req)
	}
	if !strings.Contains(updated.status, "Uninstalling remove-me") {
		t.Errorf("status = %q", updated.status)
	}
	if !updated.loading || cmd == nil {
		t.Error("expected loading state with spinner cmd")
	}
}

func TestUpdate_PluginInstalled_Success(t *testing.T) {
	m := newTestModel(t)
	m.loading = true
	m.inputActive = true
	m.input.SetValue("leftover")

	updated, req, cmd := m.Update(PluginInstalledMsg{
		Result: &plugin.InstallResult{
			Manifest: plugin.Manifest{Name: "demo"},
			Proposed: []string{"entry-a", "entry-b"},
		},
	})

	if req == nil || !req.RefreshPendingCount {
		t.Fatalf("expected RefreshPendingCount request, got %#v", req)
	}
	if updated.loading {
		t.Error("expected loading=false after install completes")
	}
	if updated.inputActive || updated.input.Value() != "" {
		t.Error("expected input cleared after install")
	}
	if !strings.Contains(updated.status, "2 entries pending approval") {
		t.Errorf("status = %q", updated.status)
	}
	if cmd == nil {
		t.Fatal("expected reload installed command")
	}
	if _, ok := cmd().(ReloadInstalledMsg); !ok {
		t.Fatalf("cmd msg type = %T, want ReloadInstalledMsg", cmd())
	}
}

func TestUpdate_PluginUninstalled_Success(t *testing.T) {
	m := newTestModel(t)
	m.loading = true

	updated, req, cmd := m.Update(PluginUninstalledMsg{
		Result: &plugin.UninstallResult{Name: "gone", Removed: []string{"a", "b"}},
	})

	if req == nil || !req.RefreshPendingCount {
		t.Fatalf("expected RefreshPendingCount request, got %#v", req)
	}
	if updated.loading {
		t.Error("expected loading=false after uninstall completes")
	}
	if !strings.Contains(updated.status, "2 entries deleted") {
		t.Errorf("status = %q", updated.status)
	}
	if cmd == nil {
		t.Fatal("expected reload installed command")
	}
}

func key(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	return tea.KeyPressMsg{Text: s}
}

func keyType(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}
