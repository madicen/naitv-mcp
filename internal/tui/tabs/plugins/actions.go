package plugins

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/store"
)

// LoadPluginsCmd loads the list of installed plugins (kind=plugin entries) from st.
func LoadPluginsCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		entries, err := st.List("plugin", nil)
		return PluginsLoadedMsg{Entries: entries, Err: err}
	}
}

// FetchRegistryCmd fetches the public plugin registry over HTTP.
func FetchRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		reg, err := plugin.LoadRegistry(plugin.DefaultRegistryURL)
		return RegistryLoadedMsg{Registry: reg, Err: err}
	}
}

// InstallPluginCmd fetches a plugin manifest from source and proposes its entries
// as pending in st. source may be a plugin name, URL, or local path.
func InstallPluginCmd(st *store.Store, source string) tea.Cmd {
	return func() tea.Msg {
		result, err := plugin.Install(st, source)
		return PluginInstalledMsg{Result: result, Err: err}
	}
}

// UninstallPluginCmd removes the named plugin and all its entries from st.
func UninstallPluginCmd(st *store.Store, name string) tea.Cmd {
	return func() tea.Msg {
		result, err := plugin.Uninstall(st, name)
		return PluginUninstalledMsg{Result: result, Err: err}
	}
}
