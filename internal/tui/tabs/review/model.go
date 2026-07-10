package review

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Request is returned from Update to signal actions the root model should handle.
type Request struct {
	ApproveSelected bool
	RejectSelected  bool
	EditSelected    bool
	ApproveAll      bool
	SwitchToEntries bool
}

// Model holds the state for the review tab.
type Model struct {
	zoneManager   *zone.Manager
	proposals     []entry.Entry
	selectedIdx   int
	width, height int
	viewport      viewport.Model
}

// NewModel creates a new review Model.
func NewModel(zm *zone.Manager) Model {
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	return Model{
		zoneManager: zm,
		viewport:    vp,
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
		if m.selectedIdx >= len(m.proposals) {
			m.selectedIdx = intMax(0, len(m.proposals)-1)
		}
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
		if m.selectedIdx >= len(m.proposals) {
			m.selectedIdx = intMax(0, len(m.proposals)-1)
		}
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
		if m.selectedIdx >= len(m.proposals) {
			m.selectedIdx = intMax(0, len(m.proposals)-1)
		}
		m.updateViewport()
		return m, nil, nil

	case AllApprovedMsg:
		m.proposals = nil
		m.selectedIdx = 0
		m.updateViewport()
		return m, nil, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.proposals)-1 {
				m.selectedIdx++
				m.updateViewport()
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.updateViewport()
			}
		case "a":
			if len(m.proposals) > 0 {
				req = &Request{ApproveSelected: true}
			}
		case "r":
			if len(m.proposals) > 0 {
				req = &Request{RejectSelected: true}
			}
		case "e":
			if len(m.proposals) > 0 {
				req = &Request{EditSelected: true}
			}
		case "A":
			if len(m.proposals) > 0 {
				req = &Request{ApproveAll: true}
			}
		case "esc":
			req = &Request{SwitchToEntries: true}
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
					m.selectedIdx = i
					m.updateViewport()
					break
				}
			}
		}
		return m, req, cmd

	case tea.MouseWheelMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		return m, req, cmd
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, req, cmd
}

// SetDimensions updates the model's dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	_, detailW := layout.SplitWidths(w)
	contentH := layout.ContentHeight(h, layout.ReviewFooterRows+2)
	vpW, vpH := layout.ViewportSize(detailW, contentH)
	// The detail viewport shares the pane with the inline action-button line.
	vpH = layout.ContentHeight(vpH, 1)
	m.viewport = viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
	m.updateViewport()
}

// SelectedProposal returns the currently selected proposal or nil.
func (m *Model) SelectedProposal() *entry.Entry {
	if len(m.proposals) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.proposals) {
		return nil
	}
	p := m.proposals[m.selectedIdx]
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
		m.viewport.SetContent("No proposals.")
		return
	}
	m.viewport.SetContent(formatProposalDetail(*p))
}

// formatProposalDetail formats a proposal for the detail pane.
func formatProposalDetail(p entry.Entry) string {
	var sb strings.Builder

	// Warn before anything else when the proposal would register a shell command.
	if tools.IsExecutable(p) {
		sb.WriteString("⚠  EXECUTABLE TOOL PROPOSAL\n")
		sb.WriteString("   Approving this will register a shell command that runs\n")
		sb.WriteString("   on the server when the model calls the tool. Review the\n")
		sb.WriteString("   exec field carefully before approving.\n\n")
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
		sb.WriteString("\nChanges proposed:\n")
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
	} else {
		if len(p.Fields) > 0 {
			sb.WriteString("\nFields:\n")
			for k, v := range p.Fields {
				sb.WriteString("  " + k + ": " + v + "\n")
			}
		}
		if p.Body != "" {
			sb.WriteString("\nBody:\n" + p.Body + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString("[ ✓ Approve (a) ]  [ ✗ Reject (r) ]  [ ✎ Edit (e) ]\n")

	return sb.String()
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
