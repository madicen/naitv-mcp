package review

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
)

// View composes the full review tab view.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	split := renderReviewSplit(m)
	actionBar := renderReviewActionBar(m)

	return strings.Join([]string{split, actionBar}, "\n")
}

// renderReviewSplit renders the left/right split pane.
func renderReviewSplit(m *Model) string {
	leftPane := renderProposalList(m, m.pane.ListW, m.pane.ContentH)
	rightPane := renderProposalDetail(m, m.pane.DetailW, m.pane.ContentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderProposalList renders the proposal list in the left pane.
func renderProposalList(m *Model, width, height int) string {
	innerW, _ := layout.ViewportSize(width, height)

	var rows []string
	for i, p := range m.proposals {
		badge := theme.BadgeNewStyle.Render("NEW")
		if p.TargetID != "" {
			badge = theme.BadgeUpdStyle.Render("UPD")
		}

		name := p.Name
		maxName := innerW - 8
		if maxName < 1 {
			maxName = 1
		}
		name = layout.Truncate(name, maxName)

		prefix := "  "
		if i == m.sel.Index {
			prefix = "▶ "
		}

		line := prefix + badge + " " + name

		rowStyle := theme.TextStyle
		if i == m.sel.Index {
			rowStyle = theme.Selected
		}
		rendered := rowStyle.Width(innerW).Render(line)
		rows = append(rows, m.zoneManager.Mark(zones.ReviewRow(i), rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, theme.DimStyle.Render("  No pending proposals"))
	}

	return listpane.RenderList(width, height, rows)
}

// renderProposalDetail renders the selected proposal detail in the right pane.
func renderProposalDetail(m *Model, width, height int) string {
	approveBtn := m.zoneManager.Mark(zones.ReviewDetailApprove, theme.ActionBtn.Render("[ ✓ Approve ]"))
	rejectBtn := m.zoneManager.Mark(zones.ReviewDetailReject, theme.ActionBtn.Render("[ ✗ Reject ]"))
	editBtn := m.zoneManager.Mark(zones.ReviewDetailEdit, theme.ActionBtn.Render("[ ✎ Edit ]"))
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, approveBtn, rejectBtn, editBtn)

	return m.detail.RenderPane(width, height, buttons)
}

// renderReviewActionBar renders the action buttons at the bottom.
func renderReviewActionBar(m *Model) string {
	return keymap.RenderActionBar(m.zoneManager, []keymap.ActionZone{
		{zones.ReviewApprove, m.keys.Approve},
		{zones.ReviewReject, m.keys.Reject},
		{zones.ReviewEdit, m.keys.Edit},
		{zones.ReviewApproveAll, m.keys.ApproveAll},
		{"", m.keys.Back},
	})
}
