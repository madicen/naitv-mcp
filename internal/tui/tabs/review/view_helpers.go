package review

import (
	"strings"

	"charm.land/lipgloss/v2"
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
	listW, detailW := layout.SplitWidths(m.width)
	contentH := layout.ContentHeight(m.height, layout.ReviewFooterRows+2)

	leftPane := renderProposalList(m, listW, contentH)
	rightPane := renderProposalDetail(m, detailW, contentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderProposalList renders the proposal list in the left pane.
func renderProposalList(m *Model, width, height int) string {
	innerW, innerH := layout.ViewportSize(width, height)

	var rows []string
	for i, p := range m.proposals {
		badge := theme.BadgeNewStyle.Render("NEW")
		if p.TargetID != "" {
			badge = theme.BadgeUpdStyle.Render("UPD")
		}

		name := p.Name
		// Budget: prefix (2) + badge (5: "NEW"/"UPD" with Padding(0,1)) + space (1).
		maxName := innerW - 8
		if maxName < 1 {
			maxName = 1
		}
		name = layout.Truncate(name, maxName)

		prefix := "  "
		if i == m.selectedIdx {
			prefix = "▶ "
		}

		line := prefix + badge + " " + name

		rowStyle := theme.TextStyle
		if i == m.selectedIdx {
			rowStyle = theme.Selected
		}
		// Pad the row to the full inner width so the entire row is a clickable
		// zone, not just the text. The Mark must wrap the padded string.
		rendered := rowStyle.Width(innerW).Render(line)

		rows = append(rows, m.zoneManager.Mark(zones.ReviewRow(i), rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, theme.DimStyle.Render("  No pending proposals"))
	}

	for len(rows) < innerH {
		rows = append(rows, "")
	}
	if len(rows) > innerH {
		rows = rows[:innerH]
	}

	content := strings.Join(rows, "\n")
	// Size the bordered box to the inner area; the rounded border adds the
	// remaining 2 rows/cols so the pane occupies exactly width × height.
	return theme.Pane.Width(innerW).Height(innerH).Render(content)
}

// renderProposalDetail renders the selected proposal detail in the right pane.
func renderProposalDetail(m *Model, width, height int) string {
	innerW, innerH := layout.ViewportSize(width, height)

	content := m.viewport.View()

	// Inline action buttons — these are also in the detail text, but add zone marks here for mouse support
	approveBtn := m.zoneManager.Mark(zones.ReviewDetailApprove, theme.ActionBtn.Render("[ ✓ Approve ]"))
	rejectBtn := m.zoneManager.Mark(zones.ReviewDetailReject, theme.ActionBtn.Render("[ ✗ Reject ]"))
	editBtn := m.zoneManager.Mark(zones.ReviewDetailEdit, theme.ActionBtn.Render("[ ✎ Edit ]"))
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, approveBtn, rejectBtn, editBtn)

	fullContent := content + "\n" + buttons
	// Size the bordered box to the inner area so the border doesn't push the
	// pane past its allotted height (which previously hid the tab bar).
	return theme.Pane.Width(innerW).Height(innerH).Render(fullContent)
}

// renderReviewActionBar renders the action buttons at the bottom.
func renderReviewActionBar(m *Model) string {
	approveBtn := m.zoneManager.Mark(zones.ReviewApprove, theme.ActionBtn.Render("[ a Approve ]"))
	rejectBtn := m.zoneManager.Mark(zones.ReviewReject, theme.ActionBtn.Render("[ r Reject ]"))
	editBtn := m.zoneManager.Mark(zones.ReviewEdit, theme.ActionBtn.Render("[ e Edit ]"))
	approveAllBtn := m.zoneManager.Mark(zones.ReviewApproveAll, theme.ActionBtn.Render("[ A Approve All ]"))
	escBtn := theme.ActionBtn.Render("[ esc Entries ]")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		approveBtn, rejectBtn, editBtn, approveAllBtn, escBtn,
	)
}
