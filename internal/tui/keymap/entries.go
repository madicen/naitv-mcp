package keymap

import "charm.land/bubbles/v2/key"

// Entries holds key bindings for the entries tab.
type Entries struct {
	Down, Up, Space, New, Edit, Delete, Delivery, Copy, Search, Review, Tab key.Binding
	ToggleMarkdown, Undo, History, Archive, Restore, Purge key.Binding
	ConfirmYes, ConfirmNo, SearchEsc key.Binding
}

// DefaultEntries is the default entries tab keymap.
var DefaultEntries = Entries{
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "down"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "up"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle group"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Delivery: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "init/ask"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Review: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "review"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "kind filter"),
	),
	ConfirmYes: key.NewBinding(
		key.WithKeys("y", "enter"),
		key.WithHelp("y", "confirm"),
	),
	ConfirmNo: key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n", "cancel"),
	),
	SearchEsc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close search"),
	),
	ToggleMarkdown: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "markdown"),
	),
	Undo: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "undo"),
	),
	History: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "history"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	Restore: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "restore"),
	),
	Purge: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "purge"),
	),
}

// EntriesActions returns bindings shown in the entries action bar.
func (k Entries) EntriesActions() []key.Binding {
	return []key.Binding{k.New, k.Edit, k.Delete, k.Delivery, k.Copy, k.Search, k.Review}
}
