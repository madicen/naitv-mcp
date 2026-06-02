package review

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleSelected  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleNormal    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	stylePane      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	styleActionBtn = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Padding(0, 1)
	styleBadgeNew  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")).Padding(0, 1)
	styleBadgeUpd  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Padding(0, 1)
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
	listW := m.width * 35 / 100
	detailW := m.width - listW - 1

	contentH := m.height - 3
	if contentH < 1 {
		contentH = 1
	}

	leftPane := renderProposalList(m, listW, contentH)
	rightPane := renderProposalDetail(m, detailW, contentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderProposalList renders the proposal list in the left pane.
func renderProposalList(m *Model, width, height int) string {
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}

	var rows []string
	for i, p := range m.proposals {
		badge := styleBadgeNew.Render("NEW")
		if p.TargetID != "" {
			badge = styleBadgeUpd.Render("UPD")
		}

		name := p.Name
		// Budget: prefix (2) + badge (5: "NEW"/"UPD" with Padding(0,1)) + space (1).
		maxName := innerW - 8
		if maxName < 1 {
			maxName = 1
		}
		if len([]rune(name)) > maxName {
			runes := []rune(name)
			name = string(runes[:maxName-1]) + "…"
		}

		prefix := "  "
		if i == m.selectedIdx {
			prefix = "▶ "
		}

		line := prefix + badge + " " + name

		rowStyle := styleNormal
		if i == m.selectedIdx {
			rowStyle = styleSelected
		}
		// Pad the row to the full inner width so the entire row is a clickable
		// zone, not just the text. The Mark must wrap the padded string.
		rendered := rowStyle.Width(innerW).Render(line)

		zoneID := proposalRowZone(i)
		rows = append(rows, m.zoneManager.Mark(zoneID, rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, styleDim.Render("  No pending proposals"))
	}

	for len(rows) < innerH {
		rows = append(rows, "")
	}
	if len(rows) > innerH {
		rows = rows[:innerH]
	}

	content := strings.Join(rows, "\n")
	return stylePane.Width(width).Height(height).Render(content)
}

// renderProposalDetail renders the selected proposal detail in the right pane.
func renderProposalDetail(m *Model, width, height int) string {
	content := m.viewport.View()

	// Inline action buttons — these are also in the detail text, but add zone marks here for mouse support
	approveBtn := m.zoneManager.Mark("detail:approve", styleActionBtn.Render("[ ✓ Approve ]"))
	rejectBtn := m.zoneManager.Mark("detail:reject", styleActionBtn.Render("[ ✗ Reject ]"))
	editBtn := m.zoneManager.Mark("detail:edit", styleActionBtn.Render("[ ✎ Edit ]"))
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, approveBtn, rejectBtn, editBtn)

	fullContent := content + "\n" + buttons
	return stylePane.Width(width).Height(height).Render(fullContent)
}

// renderReviewActionBar renders the action buttons at the bottom.
func renderReviewActionBar(m *Model) string {
	approveBtn := m.zoneManager.Mark("action:approve", styleActionBtn.Render("[ a Approve ]"))
	rejectBtn := m.zoneManager.Mark("action:reject", styleActionBtn.Render("[ r Reject ]"))
	editBtn := m.zoneManager.Mark("action:edit-review", styleActionBtn.Render("[ e Edit ]"))
	approveAllBtn := m.zoneManager.Mark("action:approve-all", styleActionBtn.Render("[ A Approve All ]"))
	escBtn := styleActionBtn.Render("[ esc Entries ]")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		approveBtn, rejectBtn, editBtn, approveAllBtn, escBtn,
	)
}
