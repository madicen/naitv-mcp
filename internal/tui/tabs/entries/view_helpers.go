package entries

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/internal/tools"
	"github.com/madicen/naitv-mcp/pkg/entry"
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
	actionBar := renderActionBar(m)

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
	trigger := m.zoneManager.Mark(zones.EntriesKindDD, m.kindDD.TriggerView())
	return theme.DimStyle.Render(kindFilterLabel) + trigger
}

// renderSplit renders the left+right split pane.
func renderSplit(m *Model) string {
	leftPane := renderList(m, m.pane.ListW, m.pane.ContentH)
	rightPane := renderDetail(m, m.pane.DetailW, m.pane.ContentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderList renders the flat entry list (with optional group headers) in the
// left pane.
func renderList(m *Model, width, height int) string {
	innerW, _ := layout.ViewportSize(width, height)

	// Groups are active when the first flat item is a header.
	hasGroups := len(m.flatItems) > 0 && m.flatItems[0].kind == itemKindHeader

	var rows []string
	for i, item := range m.flatItems {
		selected := i == m.sel.Index
		var rendered string
		if item.kind == itemKindHeader {
			rendered = renderGroupHeader(m, item, selected, innerW)
		} else {
			rendered = renderEntryRow(item.e, selected, innerW, hasGroups)
		}
		rows = append(rows, m.zoneManager.Mark(zones.EntriesRow(i), rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, theme.DimStyle.Render("  No entries"))
	}

	return listpane.RenderList(width, height, rows)
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

	count := theme.GroupCount.Render(fmt.Sprintf("(%d)", item.count))

	// Reserve: 2 (indent) + 2 (chevron + space).
	textW := width - 4
	if textW < 1 {
		textW = 1
	}
	label = layout.Truncate(label, textW)

	if selected {
		return "  " + theme.GroupHeaderSel.Render(chevron+" "+label) + " " + count
	}
	return "  " + theme.GroupHeader.Render(chevron+" "+label) + " " + count
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
		badge = theme.DimStyle.Render("[" + e.Kind + "] ")
	}
	label := e.Name
	line := badge + label
	line = layout.Truncate(line, textW)

	glyph := deliveryGlyph(e)

	if selected {
		return indent + theme.Selected.Render("▶ ") + glyph + " " + theme.Selected.Render(line)
	}
	return indent + theme.TextStyle.Render("  ") + glyph + " " + theme.TextStyle.Render(line)
}

// renderDetail renders the selected entry detail in the right pane.
func renderDetail(m *Model, width, height int) string {
	return m.detail.RenderPane(width, height)
}

// deliveryGlyph renders a styled glyph indicating an entry's delivery mode
// and kind:
//
//   - ⚡ amber  — executable tool (kind=tool with exec field); runs shell commands
//   - ● green  — init delivery (included in the initialization bundle)
//   - ○ grey   — on-demand (agent must fetch explicitly)
func deliveryGlyph(e entry.Entry) string {
	if tools.IsExecutable(e) {
		return theme.ExecToolGlyph.Render("⚡")
	}
	if e.DeliveryOrDefault() == entry.DeliveryOnDemand {
		return theme.OnDemandGlyph.Render("○")
	}
	return theme.InitGlyph.Render("●")
}

// renderActionBar renders the action buttons at the bottom.
func renderActionBar(m *Model) string {
	newBtn := m.zoneManager.Mark(zones.EntriesNew, theme.ActionBtn.Render("[ n New ]"))
	editBtn := m.zoneManager.Mark(zones.EntriesEdit, theme.ActionBtn.Render("[ e Edit ]"))
	deleteBtn := m.zoneManager.Mark(zones.EntriesDelete, theme.ActionBtn.Render("[ d Delete ]"))
	deliveryBtn := m.zoneManager.Mark(zones.EntriesDelivery, theme.ActionBtn.Render("[ i Init/Ask ]"))
	copyBtn := m.zoneManager.Mark(zones.EntriesCopy, theme.ActionBtn.Render("[ c Copy ]"))
	searchBtn := m.zoneManager.Mark(zones.EntriesSearch, theme.ActionBtn.Render("[ / Search ]"))
	reviewBtn := m.zoneManager.Mark(zones.EntriesReview, theme.ActionBtn.Render("[ R Review ]"))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		newBtn, editBtn, deleteBtn, deliveryBtn, copyBtn, searchBtn, reviewBtn,
	)
}

// renderConfirmDelete renders the delete confirmation prompt.
func renderConfirmDelete(m *Model) string {
	return theme.Confirm.Render("Delete entry? [y]es / [n]o / [esc] cancel")
}

// renderSearchBar renders the search input bar.
func renderSearchBar(m *Model) string {
	return theme.SearchBar.Render("Search: " + m.searchInput.View())
}
