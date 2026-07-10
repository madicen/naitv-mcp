package entries

import (
	"github.com/madicen/naitv-mcp/internal/tui/components/kinddropdown"
)

// refreshKindDropdown rebuilds the dropdown so its options and selection track
// the current kind set. It is a no-op while the panel is open (an open panel
// implies no structural change is in flight), mirroring appr-ai-sal's
// refreshProfileDropdown.
func (m *Model) refreshKindDropdown() {
	if m.kindDD != nil && m.kindDD.Open() {
		return
	}
	m.kindDD = kinddropdown.BuildFilter(m.zoneManager, m.kinds, m.selectedKind)
}

// kindAtIndex maps a chosen dropdown option index to a raw kind string. Index 0
// is "All" (empty string); subsequent indices map into the filtered kind set.
func (m *Model) kindAtIndex(i int) string {
	return kinddropdown.KindAtFilterIndex(i, m.kinds)
}
