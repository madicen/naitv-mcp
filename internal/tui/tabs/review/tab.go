package review

import (
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/tab"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// RequestMsg carries a review tab action for the root model to handle.
type RequestMsg struct {
	Req Request
}

// Tab wraps the review model and implements the root Tab interface.
type Tab struct {
	model Model
}

// NewTab creates a new review tab wrapper.
func NewTab(zm *zone.Manager) tab.Tab {
	return &Tab{model: NewModel(zm)}
}

// Init returns the initial command.
func (t *Tab) Init() tea.Cmd { return t.model.Init() }

// Update handles messages and emits RequestMsg commands for root-side effects.
func (t *Tab) Update(msg tea.Msg) (tab.Tab, tea.Cmd) {
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

// View renders the review tab.
func (t *Tab) View() string { return t.model.View() }

// SetDimensions updates pane dimensions.
func (t *Tab) SetDimensions(w, h int) { t.model.SetDimensions(w, h) }

// SetContentTop is a no-op for the review tab.
func (t *Tab) SetContentTop(int) {}

// InputActive reports whether a nested input owns keyboard focus.
func (t *Tab) InputActive() bool { return false }

// SelectedID returns the ID of the selected proposal.
func (t *Tab) SelectedID() string { return t.model.SelectedID() }

// SelectedProposal returns the currently selected proposal, if any.
func (t *Tab) SelectedProposal() *entry.Entry { return t.model.SelectedProposal() }
