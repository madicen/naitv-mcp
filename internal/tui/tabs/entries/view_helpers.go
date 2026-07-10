package entries

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

var (
	styleSelected       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleNormal         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleDim            = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	stylePane           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	styleActionBtn      = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Padding(0, 1)
	styleConfirm        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleSearch         = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
	styleInit           = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleOnDemand       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleExecTool       = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // amber ⚡
	styleGroupHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))  // cyan
	styleGroupHeaderSel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")) // pink (selected)
	styleGroupCount     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// kindFilterLabel prefixes the kind-filter dropdown trigger; its display width
// is the column where the trigger begins (used for SetBounds).
const kindFilterLabel = "Kind: "

// View composes the full entries tab view.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Keep the dropdown's options/selection in sync with the current kind set
	// (no-op while its panel is open).
	m.refreshKindDropdown()

	kindRow := renderKindFilter(m)
	split := renderSplit(m)
	actionBar := renderActionBar(m, m.zoneManager)

	var rows []string
	rows = append(rows, kindRow)
	rows = append(rows, split)
	if m.searchMode {
		rows = append(rows, renderSearchBar(m))
	}
	if m.showConfirmDelete {
		rows = append(rows, renderConfirmDelete(m))
	}
	rows = append(rows, actionBar)

	content := strings.Join(rows, "\n")

	// Composite the open dropdown panel over the content. The trigger sits on
	// content line 0 at the column following the label.
	if m.kindDD != nil {
		tw, th := m.kindDD.TriggerSize()
		m.kindDD.SetBounds(0, lipgloss.Width(kindFilterLabel), tw, th)
		content = m.kindDD.ViewWithOverlay(content, m.width, m.height)
	}

	return content
}

// renderKindFilter renders the kind-filter label followed by the dropdown
// trigger (marked for mouse hit-testing).
func renderKindFilter(m *Model) string {
	trigger := m.zoneManager.Mark(kindDDZone, m.kindDD.TriggerView())
	return styleDim.Render(kindFilterLabel) + trigger
}

// renderSplit renders the left+right split pane.
func renderSplit(m *Model) string {
	listW, detailW := layout.SplitWidths(m.width)
	contentH := layout.ContentHeight(m.height, layout.EntriesFooterRows+2)

	leftPane := renderList(m, listW, contentH)
	rightPane := renderDetail(m, detailW, contentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderList renders the flat entry list (with optional group headers) in the
// left pane.
func renderList(m *Model, width, height int) string {
	innerW, innerH := layout.ViewportSize(width, height)

	// Groups are active when the first flat item is a header.
	hasGroups := len(m.flatItems) > 0 && m.flatItems[0].kind == itemKindHeader

	var rows []string
	for i, item := range m.flatItems {
		selected := i == m.selectedIdx
		var rendered string
		if item.kind == itemKindHeader {
			rendered = renderGroupHeader(m, item, selected, innerW)
		} else {
			rendered = renderEntryRow(item.e, selected, innerW, hasGroups)
		}
		rows = append(rows, m.zoneManager.Mark(flatItemZone(i), rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, styleDim.Render("  No entries"))
	}

	// Pad to height.
	for len(rows) < innerH {
		rows = append(rows, "")
	}
	if len(rows) > innerH {
		rows = rows[:innerH]
	}

	content := strings.Join(rows, "\n")
	return stylePane.Width(innerW).Height(innerH).Render(content)
}

// renderGroupHeader renders one collapsible group header row.
func renderGroupHeader(m *Model, item listItem, selected bool, width int) string {
	collapsed := m.collapsed[item.groupName]
	chevron := "▼"
	if collapsed {
		chevron = "▶"
	}

	label := item.groupName
	if label == "" {
		label = "General"
	}

	count := styleGroupCount.Render(fmt.Sprintf("(%d)", item.count))

	// Reserve: 2 (indent) + 2 (chevron + space).
	textW := width - 4
	if textW < 1 {
		textW = 1
	}
	label = layout.Truncate(label, textW)

	if selected {
		return "  " + styleGroupHeaderSel.Render(chevron+" "+label) + " " + count
	}
	return "  " + styleGroupHeader.Render(chevron+" "+label) + " " + count
}

// renderEntryRow renders one entry row, optionally indented under a group header.
func renderEntryRow(e entry.Entry, selected bool, width int, indented bool) string {
	indent := ""
	if indented {
		indent = "  "
	}

	// Reserve: indent + "▶ " (2) + glyph (1) + " " (1).
	reserved := 4 + len([]rune(indent))
	textW := width - reserved
	if textW < 1 {
		textW = 1
	}

	badge := ""
	if e.Kind != "" {
		badge = styleDim.Render("[" + e.Kind + "] ")
	}
	label := e.Name
	line := badge + label
	line = layout.Truncate(line, textW)

	glyph := deliveryGlyph(e)

	if selected {
		return indent + styleSelected.Render("▶ ") + glyph + " " + styleSelected.Render(line)
	}
	return indent + styleNormal.Render("  ") + glyph + " " + styleNormal.Render(line)
}

// renderDetail renders the selected entry detail in the right pane.
func renderDetail(m *Model, width, height int) string {
	innerW, innerH := layout.ViewportSize(width, height)
	content := m.viewport.View()
	return stylePane.Width(innerW).Height(innerH).Render(content)
}

// deliveryGlyph renders a styled glyph indicating an entry's delivery mode
// and kind:
//
//   - ⚡ amber  — executable tool (kind=tool with exec field); runs shell commands
//   - ● green  — init delivery (included in the initialization bundle)
//   - ○ grey   — on-demand (agent must fetch explicitly)
func deliveryGlyph(e entry.Entry) string {
	if tools.IsExecutable(e) {
		return styleExecTool.Render("⚡")
	}
	if e.DeliveryOrDefault() == entry.DeliveryOnDemand {
		return styleOnDemand.Render("○")
	}
	return styleInit.Render("●")
}

// renderActionBar renders the action buttons at the bottom.
func renderActionBar(m *Model, zm *zone.Manager) string {
	newBtn := zm.Mark("action:new", styleActionBtn.Render("[ n New ]"))
	editBtn := zm.Mark("action:edit", styleActionBtn.Render("[ e Edit ]"))
	deleteBtn := zm.Mark("action:delete", styleActionBtn.Render("[ d Delete ]"))
	deliveryBtn := zm.Mark("action:delivery", styleActionBtn.Render("[ i Init/Ask ]"))
	copyBtn := zm.Mark("action:copy", styleActionBtn.Render("[ c Copy ]"))
	searchBtn := zm.Mark("action:search", styleActionBtn.Render("[ / Search ]"))
	reviewBtn := zm.Mark("action:review", styleActionBtn.Render("[ R Review ]"))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		newBtn, editBtn, deleteBtn, deliveryBtn, copyBtn, searchBtn, reviewBtn,
	)
}

// renderConfirmDelete renders the delete confirmation prompt.
func renderConfirmDelete(m *Model) string {
	return styleConfirm.Render("Delete entry? [y]es / [n]o / [esc] cancel")
}

// renderSearchBar renders the search input bar.
func renderSearchBar(m *Model) string {
	return styleSearch.Render("Search: " + m.searchInput.View())
}
