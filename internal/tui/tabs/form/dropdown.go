package form

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
)

// newKindOption is the sentinel option (always last) that switches the form
// into free-text "new kind" mode.
const newKindOption = "+ New kind…"

// kindLabelWidth is the rendered width of the "Kind:" label column. It must
// match theme.FormLabel's Width so the dropdown trigger's computed bounds line up
// with where the trigger is actually drawn.
const kindLabelWidth = 10

// filterKinds drops empty kinds, preserving order.
func filterKinds(kinds []string) []string {
	out := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

// displayKind capitalizes the first rune of a kind for display.
func displayKind(k string) string {
	if k == "" {
		return k
	}
	r := []rune(k)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// buildKindDropdown builds the Kind dropdown: each existing (non-empty) kind
// followed by the "+ New kind…" sentinel.
func buildKindDropdown(zm *zone.Manager, ddKinds []string) *dropdownv2.Dropdown {
	opts := make([]string, 0, len(ddKinds)+1)
	for _, k := range ddKinds {
		opts = append(opts, displayKind(k))
	}
	opts = append(opts, newKindOption)

	d := dropdownv2.New(
		dropdownv2.WithOptions(opts),
		dropdownv2.WithPlaceholder("kind"),
		dropdownv2.WithAccentColor(theme.Accent),
	)
	d.SetZoneManager(zm)
	return d
}

// SetKinds rebuilds the Kind dropdown from the given kind set and resets the
// selection to a sane default. Callers populate the specific kind afterward
// (PopulateFrom -> setKind) for edit modes.
func (m *Model) SetKinds(kinds []string) {
	m.kinds = kinds
	m.ddKinds = filterKinds(kinds)
	m.kindDD = buildKindDropdown(m.zoneManager, m.ddKinds)
	m.setKind("")
	m.syncKindFocus()
}

// setKind selects the option matching k. An existing kind selects its option
// and leaves new-kind mode; an unknown non-empty kind enters new-kind mode with
// k in the text input; an empty k selects the first existing kind (or new-kind
// mode when no kinds exist yet).
func (m *Model) setKind(k string) {
	if m.kindDD == nil {
		return
	}
	for i, kk := range m.ddKinds {
		if kk == k {
			m.newKindMode = false
			m.kindDD.SetSelectedIndex(i)
			m.kind.SetValue("")
			return
		}
	}
	if k == "" {
		if len(m.ddKinds) == 0 {
			m.newKindMode = true
		} else {
			m.newKindMode = false
		}
		m.kindDD.SetSelectedIndex(0)
		m.kind.SetValue("")
		return
	}
	m.newKindMode = true
	m.kindDD.SetSelectedIndex(len(m.ddKinds)) // sentinel index
	m.kind.SetValue(k)
}

// selectedKind returns the effective kind: the typed value in new-kind mode,
// otherwise the kind mapped from the dropdown's selected option.
func (m *Model) selectedKind() string {
	if m.newKindMode {
		return strings.TrimSpace(m.kind.Value())
	}
	if m.kindDD == nil {
		return strings.TrimSpace(m.kind.Value())
	}
	idx := m.kindDD.SelectedIndex()
	if idx >= 0 && idx < len(m.ddKinds) {
		return m.ddKinds[idx]
	}
	return strings.TrimSpace(m.kind.Value())
}

// isNewKindIndex reports whether option index i is the "+ New kind…" sentinel.
func (m *Model) isNewKindIndex(i int) bool {
	return i >= len(m.ddKinds)
}

// syncKindFocus marks the dropdown trigger focused only when the Kind field
// owns keyboard focus and the form is not in new-kind text-entry mode.
func (m *Model) syncKindFocus() {
	if m.kindDD == nil {
		return
	}
	m.kindDD.SetFocused(m.visible && m.focusIdx == 0 && !m.newKindMode)
}

// handleKindChosen applies an ItemChosenMsg: the sentinel enters new-kind mode
// (focusing the text input); any other option selects that kind.
func (m Model) handleKindChosen(msg tea.Msg) (Model, tea.Cmd) {
	cm, _ := msg.(dropdownv2.ItemChosenMsg)
	if m.kindDD != nil {
		m.kindDD, _ = m.kindDD.Update(msg)
	}
	if m.isNewKindIndex(cm.Index) {
		m.newKindMode = true
		m.kind.SetValue("")
		m.focusIdx = 0
		m.applyFocus()
	} else {
		m.newKindMode = false
		m.kind.SetValue("")
		m.kind.Blur()
	}
	m.syncKindFocus()
	return m, nil
}

// ComposeDropdownOverlay composites the form's open Kind dropdown panel onto
// mainView (the already-centered form). Bounds are derived from the centered
// form origin plus the trigger's line/column inside the panel, so the panel
// lands directly under (or above) the trigger.
func (m *Model) ComposeDropdownOverlay(mainView string, w, h int) string {
	if m.kindDD == nil || !m.kindDD.Open() {
		return mainView
	}
	modalW, modalH := overlay.ModalCellSize(m.lastFormView)
	originTop := (h - modalH) / 2
	originLeft := (w - modalW) / 2
	if originTop < 0 {
		originTop = 0
	}
	if originLeft < 0 {
		originLeft = 0
	}
	// styleFormPanel has a rounded border (1 cell) and Padding(1, 2): content
	// begins 2 rows down (border + top padding) and 3 cols in (border + left
	// padding). The trigger then sits past the "Kind:" label.
	const panelTop = 2
	const panelLeft = 3
	triggerRow := originTop + panelTop + m.kindDDRow
	triggerCol := originLeft + panelLeft + kindLabelWidth
	tw, th := m.kindDD.TriggerSize()
	m.kindDD.SetBounds(triggerRow, triggerCol, tw, th)
	return m.kindDD.ViewWithOverlay(mainView, w, h)
}
