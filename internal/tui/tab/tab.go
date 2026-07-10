package tab

import tea "charm.land/bubbletea/v2"

// Tab is a navigable pane in the root TUI (entries, review, plugins).
type Tab interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Tab, tea.Cmd)
	View() string
	SetDimensions(w, h int)
	SetContentTop(top int)
	InputActive() bool
}
