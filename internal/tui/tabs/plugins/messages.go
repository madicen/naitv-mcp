package plugins

import (
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// PluginsLoadedMsg is sent when the installed plugin list is fetched from the store.
type PluginsLoadedMsg struct {
	Entries []entry.Entry
	Err     error
}

// RegistryLoadedMsg is sent when the remote plugin registry has been fetched.
type RegistryLoadedMsg struct {
	Registry plugin.Registry
	Err      error
}

// PluginInstalledMsg is sent when an install attempt completes (success or failure).
type PluginInstalledMsg struct {
	Result *plugin.InstallResult
	Err    error
}

// PluginUninstalledMsg is sent when an uninstall attempt completes.
type PluginUninstalledMsg struct {
	Result *plugin.UninstallResult
	Err    error
}
