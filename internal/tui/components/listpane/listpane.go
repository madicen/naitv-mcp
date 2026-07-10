// Package listpane provides shared split-pane list/detail layout helpers.
package listpane

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

// Layout holds computed split-pane dimensions.
type Layout struct {
	ListW, DetailW, ContentH int
	DetailVPW, DetailVPH     int
}

// Compute returns list/detail sizes for a standard 35/65 split.
func Compute(totalW, totalH, footerOverhead, detailViewportExtra int) Layout {
	listW, detailW := layout.SplitWidths(totalW)
	contentH := layout.ContentHeight(totalH, footerOverhead)
	vpW, vpH := layout.ViewportSize(detailW, contentH)
	if detailViewportExtra > 0 {
		vpH = layout.ContentHeight(vpH, detailViewportExtra)
	}
	return Layout{
		ListW:      listW,
		DetailW:    detailW,
		ContentH:   contentH,
		DetailVPW:  vpW,
		DetailVPH:  vpH,
	}
}

// InnerSize returns inner dimensions inside a bordered pane.
func InnerSize(width, height int) (int, int) {
	return layout.ViewportSize(width, height)
}

// PadRows pads or truncates rows to fill innerH lines.
func PadRows(rows []string, innerH int) []string {
	for len(rows) < innerH {
		rows = append(rows, "")
	}
	if len(rows) > innerH {
		rows = rows[:innerH]
	}
	return rows
}

// RenderList renders rows inside a bordered list pane.
func RenderList(width, height int, rows []string) string {
	innerW, innerH := InnerSize(width, height)
	rows = PadRows(rows, innerH)
	content := strings.Join(rows, "\n")
	return theme.Pane.Width(innerW).Height(innerH).Render(content)
}

// Detail wraps a viewport for the right-hand detail pane.
type Detail struct {
	Viewport viewport.Model
}

// NewDetail creates an empty detail pane.
func NewDetail() Detail {
	return Detail{Viewport: viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))}
}

// Resize updates the detail viewport from a computed layout.
func (d *Detail) Resize(l Layout) {
	d.Viewport = viewport.New(viewport.WithWidth(l.DetailVPW), viewport.WithHeight(l.DetailVPH))
}

// SetContent replaces the viewport content.
func (d *Detail) SetContent(content string) {
	d.Viewport.SetContent(content)
}

// View returns the viewport render string.
func (d *Detail) View() string {
	return d.Viewport.View()
}

// Update forwards scroll/wheel messages to the viewport.
func (d *Detail) Update(msg tea.Msg) (Detail, tea.Cmd) {
	var cmd tea.Cmd
	d.Viewport, cmd = d.Viewport.Update(msg)
	return *d, cmd
}

// RenderPane renders the viewport inside a bordered detail pane. Extra lines are
// appended below the viewport content (e.g. inline action buttons).
func (d Detail) RenderPane(width, height int, extra ...string) string {
	innerW, innerH := InnerSize(width, height)
	content := d.Viewport.View()
	if len(extra) > 0 {
		content += "\n" + strings.Join(extra, "\n")
	}
	return theme.Pane.Width(innerW).Height(innerH).Render(content)
}

// RenderBorderless renders the viewport inside a rounded border without padding.
func (d Detail) RenderBorderless(width, height int) string {
	innerW, innerH := InnerSize(width, height)
	return theme.PluginDetail.Width(innerW).Height(innerH).Render(d.Viewport.View())
}

// Selection tracks a list cursor index.
type Selection struct {
	Index int
}

// MoveDown advances the selection when possible.
func (s *Selection) MoveDown(max int) bool {
	if max == 0 || s.Index >= max-1 {
		return false
	}
	s.Index++
	return true
}

// MoveUp retreats the selection when possible.
func (s *Selection) MoveUp() bool {
	if s.Index <= 0 {
		return false
	}
	s.Index--
	return true
}

// Clamp keeps the index in range for count items.
func (s *Selection) Clamp(count int) {
	if count == 0 {
		s.Index = 0
	} else if s.Index >= count {
		s.Index = count - 1
	}
}
