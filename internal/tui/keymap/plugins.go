package keymap

import "charm.land/bubbles/v2/key"

// Plugins holds key bindings for the plugins tab.
type Plugins struct {
	Down, Up, Tab, Refresh, Install, Uninstall, InputEsc, InputEnter key.Binding
}

// DefaultPlugins is the default plugins tab keymap.
var DefaultPlugins = Plugins{
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh registry"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Uninstall: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "uninstall"),
	),
	InputEsc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	InputEnter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
}

// PluginActions returns bindings for the installed-mode action bar.
func (k Plugins) PluginActions(installedMode bool) []key.Binding {
	bindings := []key.Binding{k.Install, k.Tab, k.Refresh}
	if installedMode {
		bindings = append(bindings[:1], append([]key.Binding{k.Uninstall}, bindings[1:]...)...)
	}
	return bindings
}
