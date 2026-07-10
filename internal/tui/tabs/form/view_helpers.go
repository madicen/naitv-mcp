package form

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
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

	inputTextW := formW - 18
	if inputTextW < 8 {
		inputTextW = 8
	}
	m.kind.SetWidth(inputTextW)

	keyTextW := formW/3 - 2
	if keyTextW < 6 {
		keyTextW = 6
	}
	valTextW := formW/2 - 2
	if valTextW < 6 {
		valTextW = 6
	}
	for i := range m.fields {
		m.fields[i].Key.SetWidth(keyTextW)
		m.fields[i].Val.SetWidth(valTextW)
	}

	var lines []string

	title := "New Entry"
	switch m.mode {
	case ModeEdit:
		title = "Edit Entry"
	case ModeEditProposal:
		title = "Edit Proposal (then Approve)"
	}
	lines = append(lines, theme.Title.Render(title))
	lines = append(lines, "")

	m.kindDDRow = len(lines)
	if m.kindDD != nil {
		trigger := m.zoneManager.Mark(zones.FormKindDD, m.kindDD.TriggerView())
		kindRow := lipgloss.JoinHorizontal(lipgloss.Center, theme.FormLabel.Render("Kind:"), trigger)
		lines = append(lines, kindRow)
		if m.newKindMode {
			lines = append(lines, renderField("New kind", m.kind.View(), m.focusIdx == 0, formW))
		}
	} else {
		lines = append(lines, renderField("Kind", m.kind.View(), m.focusIdx == 0, formW))
	}

	if m.huhForm != nil {
		lines = append(lines, m.huhForm.View())
	}

	if len(m.fields) > 0 {
		lines = append(lines, theme.FormDimLabel.Render("Custom Fields:"))
		for i, fp := range m.fields {
			keyFocused := m.focusIdx == 2+i*2
			valFocused := m.focusIdx == 2+i*2+1

			keyView := fp.Key.View()
			valView := fp.Val.View()

			var keyBox, valBox string
			if keyFocused {
				keyBox = theme.FormFocused.Width(formW / 3).Render(keyView)
			} else {
				keyBox = theme.FormInput.Width(formW / 3).Render(keyView)
			}
			if valFocused {
				valBox = theme.FormFocused.Width(formW / 2).Render(valView)
			} else {
				valBox = theme.FormInput.Width(formW / 2).Render(valView)
			}

			removeBtn := m.zoneManager.Mark(zones.FormRemoveField(i), theme.FormRemoveBtn.Render("[-]"))

			row := lipgloss.JoinHorizontal(lipgloss.Top, keyBox, " = ", valBox, " ", removeBtn)
			lines = append(lines, row)
		}
	}

	addFieldFocused := m.focusIdx == m.focusIdxAddField()
	addFieldBtn := m.zoneManager.Mark(zones.FormAddFld, renderButton("+ Add Field", addFieldFocused))
	lines = append(lines, addFieldBtn)
	lines = append(lines, "")

	saveFocused := m.focusIdx == m.focusIdxSave()
	cancelFocused := m.focusIdx == m.focusIdxCancel()
	saveBtn := m.zoneManager.Mark(zones.FormSave, renderButton("ctrl+s Save", saveFocused))
	cancelBtn := m.zoneManager.Mark(zones.FormCancel, renderButton("esc Cancel", cancelFocused))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, saveBtn, "  ", cancelBtn))

	content := strings.Join(lines, "\n")
	rendered := theme.FormPanel.Width(formW).Render(content)
	m.lastFormView = rendered
	return rendered
}

// renderField renders a labeled input field.
func renderField(label, inputView string, focused bool, width int) string {
	lbl := theme.FormLabel.Render(label + ":")
	var box string
	inputW := width - 16
	if inputW < 10 {
		inputW = 10
	}
	if focused {
		box = theme.FormFocused.Width(inputW).Render(inputView)
	} else {
		box = theme.FormInput.Width(inputW).Render(inputView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, lbl, box)
}

// renderButton renders a button, highlighted if focused.
func renderButton(label string, focused bool) string {
	if focused {
		return theme.FormBtnActive.Render(label)
	}
	return theme.FormBtn.Render(label)
}
