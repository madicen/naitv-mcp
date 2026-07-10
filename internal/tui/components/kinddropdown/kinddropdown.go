// Package kinddropdown builds kind-selection dropdowns shared by the entries
// filter and the entry form.
package kinddropdown

import (
	"strings"

	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

const (
	// FilterAll is the label for the "show all kinds" filter option.
	FilterAll = "All"
	// NewKindOption is the form sentinel that switches to free-text entry.
	NewKindOption = "+ New kind…"
	// KindLabelWidth is the rendered width of the form "Kind:" label column.
	KindLabelWidth = 10
)

// FilterKinds drops empty kinds, preserving order.
func FilterKinds(kinds []string) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

// DisplayKind capitalizes the first rune of a kind for display.
func DisplayKind(k string) string {
	if k == "" {
		return k
	}
	r := []rune(k)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// BuildFilter builds the entries kind-filter dropdown: option 0 is "All",
// followed by each non-empty kind. The trigger is kept focused.
func BuildFilter(zm *zone.Manager, kinds []string, selectedKind string) *dropdownv2.Dropdown {
	filtered := FilterKinds(kinds)

	opts := make([]string, 0, len(filtered)+1)
	opts = append(opts, FilterAll)
	for _, k := range filtered {
		opts = append(opts, DisplayKind(k))
	}

	idx := 0
	if selectedKind != "" {
		for i, k := range filtered {
			if k == selectedKind {
				idx = i + 1
				break
			}
		}
	}

	d := dropdownv2.New(
		dropdownv2.WithOptions(opts),
		dropdownv2.WithInitialIndex(idx),
		dropdownv2.WithPlaceholder(FilterAll),
		dropdownv2.WithAccentColor(theme.Accent),
	)
	d.SetZoneManager(zm)
	d.SetFocused(true)
	return d
}

// KindAtFilterIndex maps a filter dropdown option index to a raw kind string.
// Index 0 is "All" (empty string).
func KindAtFilterIndex(i int, kinds []string) string {
	if i <= 0 {
		return ""
	}
	filtered := FilterKinds(kinds)
	if i-1 < len(filtered) {
		return filtered[i-1]
	}
	return ""
}

// BuildForm builds the form Kind dropdown: each existing kind followed by the
// "+ New kind…" sentinel.
func BuildForm(zm *zone.Manager, ddKinds []string) *dropdownv2.Dropdown {
	opts := make([]string, 0, len(ddKinds)+1)
	for _, k := range ddKinds {
		opts = append(opts, DisplayKind(k))
	}
	opts = append(opts, NewKindOption)

	d := dropdownv2.New(
		dropdownv2.WithOptions(opts),
		dropdownv2.WithPlaceholder("kind"),
		dropdownv2.WithAccentColor(theme.Accent),
	)
	d.SetZoneManager(zm)
	return d
}

// IsNewKindIndex reports whether option index i is the form sentinel.
func IsNewKindIndex(i, ddKindsLen int) bool {
	return i >= ddKindsLen
}
