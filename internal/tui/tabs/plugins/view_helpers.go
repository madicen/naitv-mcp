package plugins

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

var (
	styleSelected   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleUnselected = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleMode       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Padding(0, 1)
	styleModeInact  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	styleDiv        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleDetail     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	styleList       = lipgloss.NewStyle().Padding(0, 1)
	styleStatus     = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleDim        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleBadge      = lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Bold(true) // green "installed"
	styleKey        = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleHint       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// renderList renders the left-pane list (installed or available).
func (m *Model) renderList() string {
	w := m.listW()
	h := m.contentH()

	// Mode bar
	instLabel := styleModeInact.Render("Installed")
	browLabel := styleModeInact.Render("Browse")
	if m.mode == modeInstalled {
		instLabel = styleMode.Render("Installed")
	} else {
		browLabel = styleMode.Render("Browse")
	}
	modeLine := instLabel + styleDim.Render(" | ") + browLabel

	var rows []string
	rows = append(rows, modeLine)
	rows = append(rows, styleDim.Render(strings.Repeat("─", w-2)))

	switch m.mode {
	case modeInstalled:
		if len(m.installed) == 0 {
			rows = append(rows, styleDim.Render(" No plugins installed."))
			rows = append(rows, styleDim.Render(" Press i to install one."))
		} else {
			for i, e := range m.installed {
				rows = append(rows, renderInstalledRow(e, i == m.cursor, w))
			}
		}
	case modeBrowse:
		if m.loading {
			rows = append(rows, styleDim.Render(" Fetching registry…"))
		} else if len(m.available) == 0 {
			rows = append(rows, styleDim.Render(" Press r to fetch registry."))
		} else {
			for i, re := range m.available {
				rows = append(rows, renderAvailableRow(re, i == m.cursor, w, m.installedNames[re.Name]))
			}
		}
	}

	// Pad to fill height
	for len(rows) < h {
		rows = append(rows, "")
	}
	content := strings.Join(rows[:h], "\n")
	return styleList.Width(w).Render(content)
}

// renderInstalledRow renders one row in the installed list.
func renderInstalledRow(e entry.Entry, selected bool, w int) string {
	ver := e.Fields["version"]
	cnt := e.Fields["entry_count"]
	line := fmt.Sprintf("%-22s v%-8s %s entries", truncate(e.Name, 22), ver, cnt)
	if selected {
		return styleSelected.Render("▶ " + line)
	}
	return styleUnselected.Render("  " + line)
}

// renderAvailableRow renders one row in the Browse (registry) list.
func renderAvailableRow(re plugin.RegistryEntry, selected bool, w int, installed bool) string {
	badge := ""
	if installed {
		badge = styleBadge.Render(" ✓")
	}
	line := truncate(re.Name, 24) + badge
	if selected {
		return styleSelected.Render("▶ " + line)
	}
	return styleUnselected.Render("  " + line)
}

// renderDetail renders the right-pane detail box.
func (m *Model) renderDetail() string {
	w := m.detailW()
	h := m.contentH()
	inner := styleDetail.Width(w - 2).Height(h - 2).Render(m.viewport.View())
	return inner
}

// detailContent generates the text displayed in the detail viewport.
func (m *Model) detailContent() string {
	if m.loading {
		return styleDim.Render("Working…")
	}
	switch m.mode {
	case modeInstalled:
		sel := m.selectedInstalled()
		if sel == nil {
			return styleDim.Render("No plugins installed.\nPress i to install a plugin by name, URL, or file path.")
		}
		return formatInstalledDetail(*sel)
	case modeBrowse:
		sel := m.selectedAvailable()
		if sel == nil {
			return styleDim.Render("No plugins in registry.")
		}
		return m.formatAvailableDetail(*sel)
	}
	return ""
}

// formatInstalledDetail renders the detail pane for an installed plugin.
func formatInstalledDetail(e entry.Entry) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Name:    %s\n", e.Name)
	fmt.Fprintf(&sb, "Version: %s\n", e.Fields["version"])
	if e.Fields["author"] != "" {
		fmt.Fprintf(&sb, "Author:  %s\n", e.Fields["author"])
	}
	fmt.Fprintf(&sb, "Source:  %s\n", e.Fields["source"])
	fmt.Fprintf(&sb, "Entries: %s\n", e.Fields["entry_count"])

	if e.Body != "" {
		sb.WriteString("\n")
		for _, line := range strings.Split(e.Body, "\n") {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}

	// List the entry names stored in entry_names field.
	if names := e.Fields["entry_names"]; names != "" {
		sb.WriteString("\nProposed entries:\n")
		for _, n := range strings.Split(names, ",") {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			fmt.Fprintf(&sb, "  %s\n", n)
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleDim.Render("Press u to uninstall this plugin."))
	return sb.String()
}

// formatAvailableDetail renders the detail pane for a registry plugin.
func (m *Model) formatAvailableDetail(re plugin.RegistryEntry) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Name:    %s\n", re.Name)
	fmt.Fprintf(&sb, "Version: %s\n", re.Version)
	if re.Author != "" {
		fmt.Fprintf(&sb, "Author:  %s\n", re.Author)
	}
	if len(re.Tags) > 0 {
		fmt.Fprintf(&sb, "Tags:    %s\n", strings.Join(re.Tags, ", "))
	}
	fmt.Fprintf(&sb, "URL:     %s\n", re.URL)

	if re.Description != "" {
		sb.WriteString("\n")
		// Word-wrap the description at ~60 chars.
		for _, line := range wordWrap(re.Description, 62) {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}

	sb.WriteString("\n")
	if m.installedNames[re.Name] {
		sb.WriteString(styleBadge.Render("✓ Already installed"))
	} else {
		sb.WriteString(styleKey.Render("Press i to install this plugin."))
	}
	return sb.String()
}

// renderBottom renders the action/hint bar and (when active) the input field.
func (m *Model) renderBottom() string {
	if m.inputActive {
		prompt := styleKey.Render("Install: ") + m.input.View()
		hint := styleDim.Render("  Enter to confirm · Esc to cancel")
		return prompt + hint
	}

	var hints []string
	hints = append(hints, styleKey.Render("[i]")+styleHint.Render(" install"))
	if m.mode == modeInstalled {
		hints = append(hints, styleKey.Render("[u]")+styleHint.Render(" uninstall"))
	}
	hints = append(hints, styleKey.Render("[tab]")+styleHint.Render(" switch view"))
	hints = append(hints, styleKey.Render("[r]")+styleHint.Render(" refresh registry"))

	hintLine := strings.Join(hints, styleHint.Render("  "))

	statusLine := ""
	if m.status != "" {
		statusLine = "\n" + styleStatus.Render(m.status)
	}
	return hintLine + statusLine
}

// truncate shortens s to max runes, appending … if needed.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// wordWrap breaks s into lines no longer than width runes.
func wordWrap(s string, width int) []string {
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() > 0 && cur.Len()+1+len(w) > width {
			lines = append(lines, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString(" ")
		}
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}
