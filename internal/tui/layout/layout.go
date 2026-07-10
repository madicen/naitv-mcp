// Package layout provides shared pane dimensions and text helpers for the TUI.
package layout

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	// TabBarRows is the tab bar plus the blank separator below it.
	TabBarRows = 2
	// EntriesFooterRows is the kind-filter line plus action hints.
	EntriesFooterRows = 2
	// ReviewFooterRows is the action hint line.
	ReviewFooterRows = 1
	// PluginsFooterRows is the mode line plus action hints.
	PluginsFooterRows = 2
)

// SplitWidths returns list and detail pane widths for the standard 35/65 split.
func SplitWidths(totalW int) (listW, detailW int) {
	listW = totalW * 35 / 100
	detailW = totalW - listW - 1
	if detailW < 1 {
		detailW = 1
	}
	return listW, detailW
}

// ContentHeight returns h minus fixed overhead rows, floored at 1.
func ContentHeight(h, overhead int) int {
	contentH := h - overhead
	if contentH < 1 {
		return 1
	}
	return contentH
}

// ViewportSize returns inner viewport dimensions inside a bordered pane.
func ViewportSize(paneW, paneH int) (vpW, vpH int) {
	vpW = paneW - 2
	if vpW < 1 {
		vpW = 1
	}
	vpH = paneH - 2
	if vpH < 1 {
		vpH = 1
	}
	return vpW, vpH
}

// Truncate shortens s to max display cells with an ellipsis.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	var b strings.Builder
	width := 0
	for _, r := range s {
		w := ansi.StringWidth(string(r))
		if width+w > max-1 {
			break
		}
		b.WriteRune(r)
		width += w
	}
	b.WriteString("…")
	return b.String()
}
