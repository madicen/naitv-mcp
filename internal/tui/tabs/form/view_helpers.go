package form

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	styleLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(10)
	styleInputBox  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	styleFocused   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	styleFormPanel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(1, 2)
	styleBtn       = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("39"))
	styleBtnActive = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("39")).Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("39"))
	styleRemoveBtn = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Padding(0, 1)
	styleDimLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// View renders the form as a string suitable for use as an overlay modal.
func (m *Model) View() string {
	if !m.visible {
		return ""
	}

	formW := m.width * 60 / 100
	if formW < 50 {
		formW = 50
	}
	if formW > 100 {
		formW = 100
	}

	// Size the single-line inputs to their box's inner text width so they
	// scroll horizontally instead of wrapping (which would grow the box).
	// inputW (see renderField) is formW-16; the box's Padding(0,1) leaves
	// inputW-2 cells of text. Clamp to a sane minimum.
	inputTextW := formW - 18
	if inputTextW < 8 {
		inputTextW = 8
	}
	m.kind.Width = inputTextW
	m.name.Width = inputTextW
	m.tags.Width = inputTextW
	m.body.Width = inputTextW

	// Custom field key/value boxes use formW/3 and formW/2 respectively, each
	// with Padding(0,1) → two fewer text cells.
	keyTextW := formW/3 - 2
	if keyTextW < 6 {
		keyTextW = 6
	}
	valTextW := formW/2 - 2
	if valTextW < 6 {
		valTextW = 6
	}
	for i := range m.fields {
		m.fields[i].Key.Width = keyTextW
		m.fields[i].Val.Width = valTextW
	}

	var lines []string

	// Title
	title := "New Entry"
	switch m.mode {
	case ModeEdit:
		title = "Edit Entry"
	case ModeEditProposal:
		title = "Edit Proposal (then Approve)"
	}
	lines = append(lines, styleTitle.Render(title))
	lines = append(lines, "")

	// Kind field
	lines = append(lines, renderField("Kind", m.kind.View(), m.focusIdx == 0, formW))

	// Name field
	lines = append(lines, renderField("Name", m.name.View(), m.focusIdx == 1, formW))

	// Tags field
	lines = append(lines, renderField("Tags", m.tags.View(), m.focusIdx == 2, formW))

	// Custom fields
	if len(m.fields) > 0 {
		lines = append(lines, styleDimLabel.Render("Custom Fields:"))
		for i, fp := range m.fields {
			keyFocused := m.focusIdx == 3+i*2
			valFocused := m.focusIdx == 3+i*2+1

			keyView := fp.Key.View()
			valView := fp.Val.View()

			var keyBox, valBox string
			if keyFocused {
				keyBox = styleFocused.Width(formW/3).Render(keyView)
			} else {
				keyBox = styleInputBox.Width(formW/3).Render(keyView)
			}
			if valFocused {
				valBox = styleFocused.Width(formW/2).Render(valView)
			} else {
				valBox = styleInputBox.Width(formW/2).Render(valView)
			}

			removeID := fmt.Sprintf("form:remove-field:%d", i)
			removeBtn := m.zoneManager.Mark(removeID, styleRemoveBtn.Render("[-]"))

			row := lipgloss.JoinHorizontal(lipgloss.Top, keyBox, " = ", valBox, " ", removeBtn)
			lines = append(lines, row)
		}
	}

	// Add field button
	addFieldFocused := m.focusIdx == m.focusIdxAddField()
	addFieldBtn := m.zoneManager.Mark("form:add-field", renderButton("+ Add Field", addFieldFocused))
	lines = append(lines, addFieldBtn)
	lines = append(lines, "")

	// Body field
	lines = append(lines, renderField("Body", m.body.View(), m.focusIdx == m.focusIdxBody(), formW))
	lines = append(lines, "")

	// Save / Cancel buttons
	saveFocused := m.focusIdx == m.focusIdxSave()
	cancelFocused := m.focusIdx == m.focusIdxCancel()
	saveBtn := m.zoneManager.Mark("form:save", renderButton("ctrl+s Save", saveFocused))
	cancelBtn := m.zoneManager.Mark("form:cancel", renderButton("esc Cancel", cancelFocused))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, saveBtn, "  ", cancelBtn))

	content := strings.Join(lines, "\n")
	return styleFormPanel.Width(formW).Render(content)
}

// renderField renders a labeled input field.
func renderField(label, inputView string, focused bool, width int) string {
	lbl := styleLabel.Render(label + ":")
	var box string
	// Account for the label width (10), the input box's own border (2), and
	// the form panel's horizontal padding (2+2). Otherwise the row overflows
	// the panel's inner width and lipgloss wraps the bordered box, which
	// breaks the box-drawing borders.
	inputW := width - 16
	if inputW < 10 {
		inputW = 10
	}
	if focused {
		box = styleFocused.Width(inputW).Render(inputView)
	} else {
		box = styleInputBox.Width(inputW).Render(inputView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, lbl, box)
}

// renderButton renders a button, highlighted if focused.
func renderButton(label string, focused bool) string {
	if focused {
		return styleBtnActive.Render(label)
	}
	return styleBtn.Render(label)
}
