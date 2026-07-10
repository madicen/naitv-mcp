package review

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
	"github.com/madicen/naitv-mcp/internal/tui/diff"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/markdown"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Request is returned from Update to signal actions the root model should handle.
type Request struct {
	ApproveSelected bool
	RejectSelected  bool
	EditSelected    bool
	EditBody        bool
	ApproveAll      bool
	SwitchToEntries bool
}

// Model holds the state for the review tab.
type Model struct {
	zoneManager *zone.Manager
	proposals   []entry.Entry
	targets     map[string]entry.Entry
	width, height int

	pane   listpane.Layout
	detail listpane.Detail
	sel    listpane.Selection
	keys   keymap.Review
	md     *markdown.Renderer
	renderMarkdown bool
}

// NewModel creates a new review Model.
func NewModel(zm *zone.Manager) Model {
	return Model{
		zoneManager: zm,
		detail:      listpane.NewDetail(),
		keys:        keymap.DefaultReview,
		md:          markdown.NewRenderer(80),
	}
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated model, an optional request, and a command.
func (m Model) Update(msg tea.Msg) (Model, *Request, tea.Cmd) {
	var cmd tea.Cmd
	var req *Request

	switch msg := msg.(type) {
	case ProposalsLoadedMsg:
		m.proposals = msg.Proposals
		m.sel.Clamp(len(m.proposals))
		m.updateViewport()
		return m, nil, nil

	case TargetsLoadedMsg:
		m.targets = msg.Targets
		m.updateViewport()
		return m, nil, nil

	case ProposalApprovedMsg:
		newProps := make([]entry.Entry, 0, len(m.proposals))
		for _, p := range m.proposals {
			if p.ID != msg.Entry.ID {
				newProps = append(newProps, p)
			}
		}
		m.proposals = newProps
		m.sel.Clamp(len(m.proposals))
		m.updateViewport()
		return m, nil, nil

	case ProposalRejectedMsg:
		newProps := make([]entry.Entry, 0, len(m.proposals))
		for _, p := range m.proposals {
			if p.ID != msg.ID {
				newProps = append(newProps, p)
			}
		}
		m.proposals = newProps
		m.sel.Clamp(len(m.proposals))
		m.updateViewport()
		return m, nil, nil

	case AllApprovedMsg:
		m.proposals = nil
		m.sel.Index = 0
		m.updateViewport()
		return m, nil, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Down):
			if m.sel.MoveDown(len(m.proposals)) {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Up):
			if m.sel.MoveUp() {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Approve):
			if len(m.proposals) > 0 {
				req = &Request{ApproveSelected: true}
			}
		case key.Matches(msg, m.keys.Reject):
			if len(m.proposals) > 0 {
				req = &Request{RejectSelected: true}
			}
		case key.Matches(msg, m.keys.Edit):
			if len(m.proposals) > 0 {
				req = &Request{EditSelected: true}
			}
		case key.Matches(msg, m.keys.EditBody):
			if len(m.proposals) > 0 {
				req = &Request{EditBody: true}
			}
		case key.Matches(msg, m.keys.ApproveAll):
			if len(m.proposals) > 0 {
				req = &Request{ApproveAll: true}
			}
		case key.Matches(msg, m.keys.Back):
			req = &Request{SwitchToEntries: true}
		case key.Matches(msg, m.keys.ToggleMarkdown):
			m.renderMarkdown = !m.renderMarkdown
			m.updateViewport()
		}

	case tea.MouseClickMsg:
		if m.zoneManager.Get(zones.ReviewApprove).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{ApproveSelected: true}
			}
		} else if m.zoneManager.Get(zones.ReviewReject).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{RejectSelected: true}
			}
		} else if m.zoneManager.Get(zones.ReviewEdit).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{EditSelected: true}
			}
		} else if m.zoneManager.Get(zones.ReviewApproveAll).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{ApproveAll: true}
			}
		} else if m.zoneManager.Get(zones.ReviewDetailApprove).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{ApproveSelected: true}
			}
		} else if m.zoneManager.Get(zones.ReviewDetailReject).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{RejectSelected: true}
			}
		} else if m.zoneManager.Get(zones.ReviewDetailEdit).InBounds(msg) {
			if len(m.proposals) > 0 {
				req = &Request{EditSelected: true}
			}
		} else {
			for i := range m.proposals {
				if m.zoneManager.Get(zones.ReviewRow(i)).InBounds(msg) {
					m.sel.Index = i
					m.updateViewport()
					break
				}
			}
		}
		return m, req, cmd

	case tea.MouseWheelMsg:
		m.detail, cmd = m.detail.Update(msg)
		return m, req, cmd
	}

	m.detail, cmd = m.detail.Update(msg)
	return m, req, cmd
}

// SetDimensions updates the model's dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.pane = listpane.Compute(w, h, layout.ReviewFooterRows+2, 1)
	m.detail.Resize(m.pane)
	if m.md != nil {
		m.md.SetWidth(m.pane.DetailVPW)
	}
	m.updateViewport()
}

// SelectedProposal returns the currently selected proposal or nil.
func (m *Model) SelectedProposal() *entry.Entry {
	if len(m.proposals) == 0 || m.sel.Index < 0 || m.sel.Index >= len(m.proposals) {
		return nil
	}
	p := m.proposals[m.sel.Index]
	return &p
}

// SelectedID returns the ID of the selected proposal.
func (m *Model) SelectedID() string {
	p := m.SelectedProposal()
	if p == nil {
		return ""
	}
	return p.ID
}

// updateViewport refreshes the viewport content.
func (m *Model) updateViewport() {
	p := m.SelectedProposal()
	if p == nil {
		m.detail.SetContent("No proposals.")
		return
	}
	m.detail.SetContent(m.formatProposalDetail(*p))
}

// formatProposalDetail formats a proposal for the detail pane.
func (m *Model) formatProposalDetail(p entry.Entry) string {
	var sb strings.Builder

	// Warn before anything else when the proposal would register a shell command.
	if tools.IsExecutable(p) {
		sb.WriteString("⚠  EXECUTABLE TOOL PROPOSAL\n")
		sb.WriteString("   Approving this will register a shell command that runs\n")
		sb.WriteString("   on the server when the model calls the tool. Review the\n")
		sb.WriteString("   exec field carefully before approving.\n")
		if def, err := tools.ParseDef(p); err == nil {
			fmt.Fprintf(&sb, "\nCommand: %s\n", tools.ShellCommandLine(def, nil))
		}
		sb.WriteString("\n")
	}

	badge := "NEW"
	if p.TargetID != "" {
		badge = "UPD"
	}

	fmt.Fprintf(&sb, "[%s] %s\n\n", badge, p.Name)
	sb.WriteString("Kind:  " + p.Kind + "\n")
	sb.WriteString("ID:    " + p.ID + "\n")
	if p.ProposedBy != "" {
		sb.WriteString("By:    " + p.ProposedBy + "\n")
	}
	if p.ProposedAt != nil {
		sb.WriteString("At:    " + p.ProposedAt.Format("2006-01-02 15:04:05") + "\n")
	}
	if len(p.Tags) > 0 {
		sb.WriteString("Tags:  " + strings.Join(p.Tags, ", ") + "\n")
	}

	if p.TargetID != "" {
		sb.WriteString("\nTarget ID: " + p.TargetID + "\n")
		sb.WriteString("\nChanges proposed:\n\n")
		if target, ok := m.targets[p.TargetID]; ok {
			if p.Name != "" && p.Name != target.Name {
				sb.WriteString(diff.Unified("name", target.Name, p.Name))
				sb.WriteString("\n\n")
			}
			if p.Kind != "" && p.Kind != target.Kind {
				sb.WriteString(diff.Unified("kind", target.Kind, p.Kind))
				sb.WriteString("\n\n")
			}
			if p.Body != "" && p.Body != target.Body {
				sb.WriteString(diff.Unified("body", target.Body, p.Body))
				sb.WriteString("\n\n")
			}
			if len(p.Tags) > 0 {
				oldTags := strings.Join(target.Tags, ", ")
				newTags := strings.Join(p.Tags, ", ")
				if oldTags != newTags {
					sb.WriteString(diff.Unified("tags", oldTags, newTags))
					sb.WriteString("\n\n")
				}
			}
			if fieldsDiff := diff.FieldsDiff(target.Fields, p.Fields); fieldsDiff != "" {
				sb.WriteString(fieldsDiff)
			}
		} else {
			if p.Name != "" {
				sb.WriteString("  ~ name → " + p.Name + "\n")
			}
			if p.Body != "" {
				sb.WriteString("  ~ body → " + p.Body + "\n")
			}
			if p.Kind != "" {
				sb.WriteString("  ~ kind → " + p.Kind + "\n")
			}
			if len(p.Tags) > 0 {
				sb.WriteString("  ~ tags → " + strings.Join(p.Tags, ", ") + "\n")
			}
			for k, v := range p.Fields {
				sb.WriteString("  ~ " + k + " → " + v + "\n")
			}
		}
	} else {
		if len(p.Fields) > 0 {
			sb.WriteString("\nFields:\n")
			for k, v := range p.Fields {
				sb.WriteString("  " + k + ": " + v + "\n")
			}
		}
		if p.Body != "" {
			sb.WriteString("\nBody")
			if m.renderMarkdown {
				sb.WriteString(" (rendered, m=toggle):\n")
				sb.WriteString(m.md.RenderBody(p.ID, p.Body))
			} else {
				sb.WriteString(" (raw, m=toggle):\n" + p.Body + "\n")
			}
		}
	}

	return sb.String()
}
