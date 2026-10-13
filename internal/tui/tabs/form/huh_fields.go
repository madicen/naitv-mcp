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

const (
	focusName  = 1
	focusGroup = 2
	focusTags  = 3
	focusBody  = 4
	huhFieldN  = 4 // name, group, tags, body
)

type huhFields struct {
	name  string
	group string
	tags  string
	body  string
}

func (m *Model) rebuildHuhForm() {
	if m.huhVals == nil {
		m.huhVals = &huhFields{}
	}
	fields := m.huhVals
	m.huhName = huh.NewInput().
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
		})
	m.huhGroup = huh.NewInput().
		Key("group").
		Title("Group").
		Placeholder("group (optional)").
		CharLimit(100).
		Value(&fields.group)
	m.huhTags = huh.NewInput().
		Key("tags").
		Title("Tags").
		Placeholder("tags (comma-separated)").
		CharLimit(500).
		Value(&fields.tags)
	m.huhBody = huh.NewText().
		Key("body").
		Title("Body").
		Placeholder("body / description").
		CharLimit(5000).
		Lines(4).
		Value(&fields.body)

	m.huhForm = huh.NewForm(
		huh.NewGroup(m.huhName, m.huhGroup, m.huhTags, m.huhBody),
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
	if m.huhVals == nil {
		m.huhVals = &huhFields{}
	}
	m.huhVals.name = e.Name
	m.huhVals.group = e.Group
	m.huhVals.tags = strings.Join(e.Tags, ", ")
	m.huhVals.body = e.Body
	m.rebuildHuhForm()
}

func (m *Model) clearHuhFields() {
	if m.huhVals == nil {
		m.huhVals = &huhFields{}
	} else {
		*m.huhVals = huhFields{}
	}
	m.rebuildHuhForm()
}

func (m *Model) huhField(idx int) huh.Field {
	switch idx {
	case focusName:
		return m.huhName
	case focusGroup:
		return m.huhGroup
	case focusTags:
		return m.huhTags
	case focusBody:
		return m.huhBody
	default:
		return nil
	}
}

func (m *Model) blurHuh() {
	for _, f := range []huh.Field{m.huhName, m.huhGroup, m.huhTags, m.huhBody} {
		if f != nil {
			f.Blur()
		}
	}
}

// activateHuh runs huh init synchronously so fields are sized and themed.
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

// ensureHuhReady builds huh field state after rebuild. The root model swallows
// tea.WindowSizeMsg after SetDimensions, so without Init the fields stay empty.
func (m *Model) ensureHuhReady() {
	if m.huhForm == nil {
		return
	}
	m.activateHuh()
	m.blurHuh()
}

// naitvTheme maps huh styles to the shared TUI theme colors.
type naitvTheme struct{}

func (naitvTheme) Theme(_ bool) *huh.Styles {
	s := huh.ThemeCharm(false)
	text := lipgloss.Color(theme.Text)
	accent := lipgloss.Color(theme.Accent)
	// theme.Dim (240) is invisible on the dark background — Kind uses Text.
	s.Focused.Title = s.Focused.Title.Foreground(accent)
	s.Blurred.Title = s.Blurred.Title.Foreground(text)
	s.Focused.Base = s.Focused.Base.BorderForeground(accent)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	value := lipgloss.NewStyle().Foreground(text)
	s.Focused.TextInput.Placeholder = hint
	s.Blurred.TextInput.Placeholder = hint
	s.Focused.TextInput.Text = value
	s.Blurred.TextInput.Text = value
	s.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(accent)
	s.Blurred.TextInput.Prompt = lipgloss.NewStyle().Foreground(text)
	return s
}
