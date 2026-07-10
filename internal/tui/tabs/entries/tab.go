package entries

import (
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tab"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// RequestMsg carries an entries tab action for the root model to handle.
type RequestMsg struct {
	Req Request
}

// Tab wraps the entries model and implements the root Tab interface.
type Tab struct {
	model Model
}

// NewTab creates a new entries tab wrapper.
func NewTab(zm *zone.Manager) tab.Tab {
	return &Tab{model: NewModel(zm)}
}

// Init returns the initial command.
func (t *Tab) Init() tea.Cmd { return t.model.Init() }

// Update handles messages and emits RequestMsg commands for root-side effects.
func (t *Tab) Update(msg tea.Msg) (tab.Tab, tea.Cmd) {
	switch msg.(type) {
	case dropdownv2.ItemChosenMsg, dropdownv2.ItemCanceledMsg:
		return t.updateModel(msg)
	}
	return t.updateModel(msg)
}

func (t *Tab) updateModel(msg tea.Msg) (tab.Tab, tea.Cmd) {
	m, req, cmd := t.model.Update(msg)
	t.model = m
	return t, tea.Batch(cmd, requestCmd(req))
}

func requestCmd(req *Request) tea.Cmd {
	if req == nil {
		return nil
	}
	r := *req
	return func() tea.Msg { return RequestMsg{Req: r} }
}

// View renders the entries tab.
func (t *Tab) View() string { return t.model.View() }

// SetDimensions updates pane dimensions.
func (t *Tab) SetDimensions(w, h int) { t.model.SetDimensions(w, h) }

// SetContentTop records the absolute row where tab content begins.
func (t *Tab) SetContentTop(top int) { t.model.SetContentTop(top) }

// InputActive reports whether a nested input owns keyboard focus.
func (t *Tab) InputActive() bool { return false }

// SelectedKind returns the active kind filter.
func (t *Tab) SelectedKind() string { return t.model.SelectedKind() }

// SetSelectedKind sets the active kind filter.
func (t *Tab) SetSelectedKind(kind string) { t.model.SetSelectedKind(kind) }

// Kinds returns distinct kinds known to the entries tab.
func (t *Tab) Kinds() []string { return t.model.Kinds() }

// SelectedEntry returns the currently selected entry, if any.
func (t *Tab) SelectedEntry() *entry.Entry { return t.model.SelectedEntry() }

// DeleteTargetID returns the ID of the entry pending deletion.
func (t *Tab) DeleteTargetID() string { return t.model.DeleteTargetID() }

// SearchQuery returns the current search query.
func (t *Tab) SearchQuery() string { return t.model.SearchQuery() }

// ShowArchived reports whether the archive filter is active.
func (t *Tab) ShowArchived() bool { return t.model.ShowArchived() }

// SelectedHistoryID returns the selected history record ID.
func (t *Tab) SelectedHistoryID() string { return t.model.SelectedHistoryID() }
