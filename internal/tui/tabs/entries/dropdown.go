package entries

import (
	"strings"

	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

// kindDDZone is the bubblezone ID for the kind-filter dropdown trigger.
const kindDDZone = "entries:kind-dd"

// kindFilterAll is the label for the "show all kinds" option (always index 0).
const kindFilterAll = "All"

// filterKinds drops empty kinds, preserving order. The result is the canonical
// slice used both to build the dropdown options and to map a chosen option
// index back to a raw kind string (so the two never drift).
func filterKinds(kinds []string) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

// displayKind capitalizes the first rune of a kind for display, matching the
// label style the old pill row used.
func displayKind(k string) string {
	if k == "" {
		return k
	}
	r := []rune(k)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// newKindDropdown builds the kind-filter dropdown: option 0 is "All", followed
// by each non-empty kind (capitalized for display). The selection is derived
// from selectedKind, and the trigger is kept focused so its accent arrow shows
// and Enter/Space (or a synthesized Enter on Tab) opens it.
func newKindDropdown(zm *zone.Manager, kinds []string, selectedKind string) *dropdownv2.Dropdown {
	filtered := filterKinds(kinds)

	opts := make([]string, 0, len(filtered)+1)
	opts = append(opts, kindFilterAll)
	for _, k := range filtered {
		opts = append(opts, displayKind(k))
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
		dropdownv2.WithPlaceholder(kindFilterAll),
		dropdownv2.WithAccentColor(theme.Accent),
	)
	d.SetZoneManager(zm)
	d.SetFocused(true)
	return d
}

// refreshKindDropdown rebuilds the dropdown so its options and selection track
// the current kind set. It is a no-op while the panel is open (an open panel
// implies no structural change is in flight), mirroring appr-ai-sal's
// refreshProfileDropdown.
func (m *Model) refreshKindDropdown() {
	if m.kindDD != nil && m.kindDD.Open() {
		return
	}
	m.kindDD = newKindDropdown(m.zoneManager, m.kinds, m.selectedKind)
}

// kindAtIndex maps a chosen dropdown option index to a raw kind string. Index 0
// is "All" (empty string); subsequent indices map into the filtered kind set.
func (m *Model) kindAtIndex(i int) string {
	if i <= 0 {
		return ""
	}
	filtered := filterKinds(m.kinds)
	if i-1 < len(filtered) {
		return filtered[i-1]
	}
	return ""
}
