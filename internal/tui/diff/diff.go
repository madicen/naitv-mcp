// Package diff renders styled unified diffs for the TUI.
package diff

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/aymanbagabas/go-udiff"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

var (
	addStyle  = lipgloss.NewStyle().Background(lipgloss.Color("22")).Foreground(lipgloss.Color("255"))
	delStyle  = lipgloss.NewStyle().Background(lipgloss.Color("52")).Foreground(lipgloss.Color("255"))
	metaStyle = theme.DimStyle
)

func Unified(label, oldText, newText string) string {
	if oldText == newText {
		return metaStyle.Render("(no changes)")
	}
	raw := udiff.Unified("current", "proposed", oldText, newText)
	if strings.TrimSpace(raw) == "" {
		return metaStyle.Render("(no changes)")
	}
	if label != "" {
		return metaStyle.Render(label+":") + "\n" + styleUnified(raw)
	}
	return styleUnified(raw)
}

func styleUnified(raw string) string {
	var sb strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "+"):
			sb.WriteString(addStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			sb.WriteString(delStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			sb.WriteString(metaStyle.Render(line))
		default:
			sb.WriteString(line)
		}
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

func FieldsDiff(oldFields, newFields map[string]string) string {
	if len(newFields) == 0 {
		return ""
	}
	var parts []string
	for k, newVal := range newFields {
		oldVal := oldFields[k]
		if oldVal == newVal {
			continue
		}
		parts = append(parts, Unified(fmt.Sprintf("field %s", k), oldVal, newVal))
	}
	return strings.Join(parts, "\n\n")
}
