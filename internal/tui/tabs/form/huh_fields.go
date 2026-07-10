package form

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

type huhFields struct {
	name string
	group string
	tags string
	body string
}

func (m *Model) rebuildHuhForm() {
	fields := &m.huhVals
	m.huhForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Name").
				Placeholder("name").
				CharLimit(200).
				Value(&fields.name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Key("group").
				Title("Group").
				Placeholder("group (optional)").
				CharLimit(100).
				Value(&fields.group),
			huh.NewInput().
				Key("tags").
				Title("Tags").
				Placeholder("tags (comma-separated)").
				CharLimit(500).
				Value(&fields.tags),
			huh.NewText().
				Key("body").
				Title("Body").
				Placeholder("body / description").
				CharLimit(5000).
				Lines(4).
				Value(&fields.body),
		),
	).WithTheme(naitvTheme{}).WithShowHelp(false)
	if m.width > 0 {
		formW := m.width * 60 / 100
		if formW < 50 {
			formW = 50
		}
		if formW > 100 {
			formW = 100
		}
		m.huhForm = m.huhForm.WithWidth(formW - 6)
	}
}

func (m *Model) syncHuhFromEntry(e entry.Entry) {
	m.huhVals.name = e.Name
	m.huhVals.group = e.Group
	m.huhVals.tags = strings.Join(e.Tags, ", ")
	m.huhVals.body = e.Body
	m.rebuildHuhForm()
}

func (m *Model) clearHuhFields() {
	m.huhVals = huhFields{}
	m.rebuildHuhForm()
}

// activateHuh runs huh init synchronously so key input works immediately.
func (m *Model) activateHuh() {
	if m.huhForm == nil {
		return
	}
	cmd := m.huhForm.Init()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		var next tea.Cmd
		updated, next := m.huhForm.Update(msg)
		if f, ok := updated.(*huh.Form); ok {
			m.huhForm = f
		}
		cmd = next
	}
	if m.width > 0 {
		size := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		updated, _ := m.huhForm.Update(size)
		if f, ok := updated.(*huh.Form); ok {
			m.huhForm = f
		}
	}
}

// naitvTheme maps huh styles to the shared TUI theme colors.
type naitvTheme struct{}

func (naitvTheme) Theme(_ bool) *huh.Styles {
	s := huh.ThemeCharm(false)
	s.Focused.Title = s.Focused.Title.Foreground(lipgloss.Color(theme.Accent))
	s.Blurred.Title = s.Blurred.Title.Foreground(lipgloss.Color(theme.Dim))
	return s
}
