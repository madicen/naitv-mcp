package plugins

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// renderList renders the left-pane list (installed or available).
func (m *Model) renderList() string {
	w := m.listW()
	h := m.contentH()

	// Mode bar
	instLabel := theme.PluginModeInact.Render("Installed")
	browLabel := theme.PluginModeInact.Render("Browse")
	if m.mode == modeInstalled {
		instLabel = theme.PluginMode.Render("Installed")
	} else {
		browLabel = theme.PluginMode.Render("Browse")
	}
	modeLine := instLabel + theme.DimStyle.Render(" | ") + browLabel

	var rows []string
	rows = append(rows, modeLine)
	rows = append(rows, theme.DimStyle.Render(strings.Repeat("─", w-2)))

	switch m.mode {
	case modeInstalled:
		if len(m.installed) == 0 {
			rows = append(rows, theme.DimStyle.Render(" No plugins installed."))
			rows = append(rows, theme.DimStyle.Render(" Press i to install one."))
		} else {
			for i, e := range m.installed {
				rows = append(rows, renderInstalledRow(e, i == m.sel.Index, w))
			}
		}
	case modeBrowse:
		if m.loading {
			rows = append(rows, m.spin.View()+theme.DimStyle.Render(" Fetching registry…"))
		} else if len(m.available) == 0 {
			rows = append(rows, theme.DimStyle.Render(" Press r to fetch registry."))
		} else {
			for i, re := range m.available {
				rows = append(rows, renderAvailableRow(re, i == m.sel.Index, w, m.installedNames[re.Name]))
			}
		}
	}

	// Pad to fill height
	for len(rows) < h {
		rows = append(rows, "")
	}
	content := strings.Join(rows[:h], "\n")
	return theme.PluginList.Width(w).Render(content)
}

// renderInstalledRow renders one row in the installed list.
func renderInstalledRow(e entry.Entry, selected bool, w int) string {
	ver := e.Fields["version"]
	cnt := e.Fields["entry_count"]
	line := fmt.Sprintf("%-22s v%-8s %s entries", layout.Truncate(e.Name, 22), ver, cnt)
	if selected {
		return theme.Selected.Render("▶ " + line)
	}
	return theme.TextStyle.Render("  " + line)
}

// renderAvailableRow renders one row in the Browse (registry) list.
func renderAvailableRow(re plugin.RegistryEntry, selected bool, w int, installed bool) string {
	badge := ""
	if installed {
		badge = theme.PluginInstalled.Render(" ✓")
	}
	line := layout.Truncate(re.Name, 24) + badge
	if selected {
		return theme.Selected.Render("▶ " + line)
	}
	return theme.TextStyle.Render("  " + line)
}

// renderDetail renders the right-pane detail box.
func (m *Model) renderDetail() string {
	return m.detail.RenderBorderless(m.detailW(), m.contentH())
}

// detailContent generates the text displayed in the detail viewport.
func (m *Model) detailContent() string {
	if m.loading {
		return m.spin.View() + theme.DimStyle.Render(" Working…")
	}
	switch m.mode {
	case modeInstalled:
		sel := m.selectedInstalled()
		if sel == nil {
			return theme.DimStyle.Render("No plugins installed.\nPress i to install a plugin by name, URL, or file path.")
		}
		return formatInstalledDetail(*sel)
	case modeBrowse:
		sel := m.selectedAvailable()
		if sel == nil {
			return theme.DimStyle.Render("No plugins in registry.")
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

	// List linked entry IDs from the tracker.
	if raw := e.Fields["entry_ids"]; raw != "" {
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
			sb.WriteString("\nLinked entries:\n")
			for _, id := range ids {
				fmt.Fprintf(&sb, "  %s\n", id)
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(theme.DimStyle.Render("Press u to uninstall this plugin."))
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
		sb.WriteString(theme.PluginInstalled.Render("✓ Already installed"))
	} else {
		sb.WriteString(theme.KeyHint.Render("Press i to install this plugin."))
	}
	return sb.String()
}

// renderBottom renders the action/hint bar and (when active) the input field.
func (m *Model) renderBottom() string {
	if m.inputActive {
		prompt := theme.KeyHint.Render("Install: ") + m.input.View()
		hint := theme.Hint.Render("  Enter to confirm · Esc to cancel")
		return prompt + hint
	}

	var hints []keymap.ActionZone
	hints = append(hints, keymap.ActionZone{ZoneID: zones.PluginActInstall, Binding: m.keys.Install})
	if m.mode == modeInstalled {
		hints = append(hints, keymap.ActionZone{ZoneID: zones.PluginActUninstall, Binding: m.keys.Uninstall})
	}
	hints = append(hints,
		keymap.ActionZone{ZoneID: zones.PluginActTab, Binding: m.keys.Tab},
		keymap.ActionZone{ZoneID: zones.PluginActRefresh, Binding: m.keys.Refresh},
	)
	hintLine := keymap.RenderActionBar(m.zoneManager, hints)

	statusLine := ""
	if m.status != "" {
		statusLine = "\n" + theme.StatusStyle.Render(m.status)
	}
	return hintLine + statusLine
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
