package entries

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

var (
	styleTabActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	styleTabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	styleSelected    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleNormal      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleDim         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	stylePane        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	styleActionBtn   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Padding(0, 1)
	styleConfirm     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleSearch      = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
	styleInit        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleOnDemand    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// View composes the full entries tab view.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	kindTabsRow := renderKindTabs(m)
	split := renderSplit(m)
	actionBar := renderActionBar(m, m.zoneManager)

	var rows []string
	rows = append(rows, kindTabsRow)
	rows = append(rows, split)
	if m.searchMode {
		rows = append(rows, renderSearchBar(m))
	}
	if m.showConfirmDelete {
		rows = append(rows, renderConfirmDelete(m))
	}
	rows = append(rows, actionBar)

	return strings.Join(rows, "\n")
}

// renderKindTabs renders the kind filter pill row.
func renderKindTabs(m *Model) string {
	allLabel := "All"
	var pills []string

	if m.selectedKind == "" {
		pills = append(pills, m.zoneManager.Mark("kind:", styleTabActive.Render(allLabel)))
	} else {
		pills = append(pills, m.zoneManager.Mark("kind:", styleTabInactive.Render(allLabel)))
	}

	for _, k := range m.kinds {
		if len(k) == 0 {
			continue
		}
		label := strings.ToUpper(k[:1]) + k[1:]
		id := "kind:" + k
		if m.selectedKind == k {
			pills = append(pills, m.zoneManager.Mark(id, styleTabActive.Render(label)))
		} else {
			pills = append(pills, m.zoneManager.Mark(id, styleTabInactive.Render(label)))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, pills...)
}

// renderSplit renders the left+right split pane.
func renderSplit(m *Model) string {
	listW := m.width * 35 / 100
	detailW := m.width - listW - 1

	contentH := m.height - 4
	if contentH < 1 {
		contentH = 1
	}

	leftPane := renderList(m, listW, contentH)
	rightPane := renderDetail(m, detailW, contentH)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

// renderList renders the entry list in the left pane.
func renderList(m *Model, width, height int) string {
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}

	// Reserve room for the selection arrow + delivery glyph (e.g. "▶ ● ").
	textW := innerW - 4
	if textW < 1 {
		textW = 1
	}

	var rows []string
	for i, e := range m.entries {
		label := e.Name
		badge := ""
		if e.Kind != "" {
			badge = styleDim.Render("[" + e.Kind + "] ")
		}

		line := badge + label
		if len([]rune(line)) > textW {
			runes := []rune(line)
			line = string(runes[:textW-1]) + "…"
		}

		glyph := deliveryGlyph(e)

		var prefix, rendered string
		if i == m.selectedIdx {
			prefix = styleSelected.Render("▶ ")
			rendered = prefix + glyph + " " + styleSelected.Render(line)
		} else {
			prefix = styleNormal.Render("  ")
			rendered = prefix + glyph + " " + styleNormal.Render(line)
		}

		zoneID := fmt.Sprintf("entry:%d", i)
		rows = append(rows, m.zoneManager.Mark(zoneID, rendered))
	}

	if len(rows) == 0 {
		rows = append(rows, styleDim.Render("  No entries"))
	}

	// Pad to height
	for len(rows) < innerH {
		rows = append(rows, "")
	}
	if len(rows) > innerH {
		rows = rows[:innerH]
	}

	content := strings.Join(rows, "\n")
	return stylePane.Width(innerW).Height(innerH).Render(content)
}

// renderDetail renders the selected entry detail in the right pane.
func renderDetail(m *Model, width, height int) string {
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	content := m.viewport.View()
	return stylePane.Width(innerW).Height(innerH).Render(content)
}

// deliveryGlyph renders a styled glyph indicating an entry's delivery mode:
// a filled dot for init (always loaded), a hollow dot for on-demand.
func deliveryGlyph(e entry.Entry) string {
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
	searchBtn := zm.Mark("action:search", styleActionBtn.Render("[ / Search ]"))
	reviewBtn := zm.Mark("action:review", styleActionBtn.Render("[ R Review ]"))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		newBtn, editBtn, deleteBtn, deliveryBtn, searchBtn, reviewBtn,
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
