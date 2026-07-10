package keymap

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

// ActionZone pairs a zone ID with a key binding for clickable action bars.
type ActionZone struct {
	ZoneID  string
	Binding key.Binding
}

// RenderActionBar renders zone-marked action hints from bindings.
func RenderActionBar(zm *zone.Manager, actions []ActionZone) string {
	if len(actions) == 0 {
		return ""
	}
	parts := make([]string, len(actions))
	for i, a := range actions {
		label := theme.KeyHint.Render("["+a.Binding.Help().Key+"]") +
			theme.Hint.Render(" "+a.Binding.Help().Desc)
		if zm != nil && a.ZoneID != "" {
			label = zm.Mark(a.ZoneID, label)
		}
		parts[i] = label
	}
	return strings.Join(parts, theme.Hint.Render("  "))
}

// RenderHelp renders a short help line from bindings without zones.
func RenderHelp(h help.Model, bindings []key.Binding) string {
	h.SetWidth(120)
	return h.ShortHelpView(bindings)
}
