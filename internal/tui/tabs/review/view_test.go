package review

import (
	"testing"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// TestViewFitsHeight guards against the regression where the review panes were
// sized with the full width/height (so the rounded border pushed the view 2
// rows taller than its budget, scrolling the root tab bar off-screen).
func TestViewFitsHeight(t *testing.T) {
	for _, h := range []int{10, 24, 50} {
		m := NewModel(zone.New())
		m.SetDimensions(80, h)
		got := lipgloss.Height(m.View())
		if got > h {
			t.Errorf("h=%d: view height %d exceeds budget %d (would hide the tab bar)", h, got, h)
		}
	}
}
