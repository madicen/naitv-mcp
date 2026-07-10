package form

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/kinddropdown"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Mode represents the form operation mode.
type Mode int

const (
	ModeCreate Mode = iota
	ModeEdit
	ModeEditProposal
)

// FieldPair is a key/value pair for custom fields.
type FieldPair struct {
	Key textinput.Model
	Val textinput.Model
}

// SaveMsg is emitted when the form is saved.
type SaveMsg struct {
	E          entry.Entry
	ProposalID string // non-empty means edit-before-approve
}

// CancelMsg is emitted when the form is cancelled.
type CancelMsg struct{}

// Model holds the form state.
type Model struct {
	mode          Mode
	visible       bool
	focusIdx      int
	kind          textinput.Model
	fields        []FieldPair
	// Pass-through fields: not exposed in the form UI but must survive
	// PopulateFrom → ToEntry so store.Update doesn't zero them out.
	sourceID         string
	sourceStatus     entry.Status
	sourceProposedBy string
	sourceProposedAt *time.Time
	sourceTargetID   string
	proposalID       string
	delivery         entry.Delivery
	width, height int
	zoneManager   *zone.Manager

	// Kind selection: kindDD is a dropdown of existing kinds plus a
	// "+ New kind…" sentinel. When newKindMode is set the kind textinput is
	// revealed for free-text entry. kinds is the raw set; ddKinds is the
	// non-empty subset used for the option-index ↔ kind mapping.
	kindDD      *dropdownv2.Dropdown
	kinds       []string
	ddKinds     []string
	newKindMode bool
	keys        keymap.Form
	huhForm     *huh.Form
	huhVals     huhFields
	// kindDDRow is the content-line index of the trigger within the form panel
	// body; lastFormView is the most recent rendered panel. Both are set in
	// View and consumed by ComposeDropdownOverlay.
	kindDDRow    int
	lastFormView string
}

// fieldCount returns the total number of focusable items.
// Indices: 0=kind, 1=huh fields, 2..2+len*2-1=custom fields, addField, save, cancel
func (m *Model) fieldCount() int {
	return 2 + len(m.fields)*2 + 1 + 2
}

func (m *Model) focusIdxAddField() int { return 2 + len(m.fields)*2 }
func (m *Model) focusIdxSave() int     { return m.focusIdxAddField() + 1 }
func (m *Model) focusIdxCancel() int   { return m.focusIdxSave() + 1 }

// NewModel creates a new form Model.
func NewModel(zm *zone.Manager) Model {
	kind := textinput.New()
	kind.Placeholder = "kind (e.g. rule, fact, instruction)"
	kind.CharLimit = 100

	m := Model{
		kind:        kind,
		zoneManager: zm,
		keys:        keymap.DefaultForm,
	}
	// Start with an empty kind dropdown (only the sentinel); SetKinds rebuilds
	// it with the real kind set whenever the form is opened.
	m.kindDD = kinddropdown.BuildForm(zm, nil)
	m.setKind("")
	m.clearHuhFields()
	return m
}

// Visible returns true if the form is visible.
func (m *Model) Visible() bool { return m.visible }

// Show makes the form visible and focuses the first field.
func (m *Model) Show() {
	m.visible = true
	m.focusIdx = 0
	m.applyFocus()
}

// Hide hides the form.
func (m *Model) Hide() {
	m.visible = false
	m.blurAll()
}

// SetMode sets the form mode.
func (m *Model) SetMode(mode Mode) { m.mode = mode }

// GetMode returns the current mode.
func (m *Model) GetMode() Mode { return m.mode }

// PopulateFrom fills the form from an entry.
func (m *Model) PopulateFrom(e entry.Entry) {
	m.setKind(e.Kind)
	m.syncHuhFromEntry(e)
	m.sourceID = e.ID
	m.sourceStatus = e.Status
	m.sourceProposedBy = e.ProposedBy
	m.sourceProposedAt = e.ProposedAt
	m.sourceTargetID = e.TargetID
	m.delivery = e.DeliveryOrDefault()

	m.fields = nil
	for k, v := range e.Fields {
		kInput := textinput.New()
		kInput.Placeholder = "field name"
		kInput.CharLimit = 100
		kInput.SetValue(k)

		vInput := textinput.New()
		vInput.Placeholder = "field value"
		vInput.CharLimit = 1000
		vInput.SetValue(v)

		m.fields = append(m.fields, FieldPair{Key: kInput, Val: vInput})
	}
}

// SetProposalID sets the proposal ID for ModeEditProposal.
func (m *Model) SetProposalID(id string) { m.proposalID = id }

// Reset clears all form fields.
func (m *Model) Reset() {
	m.kind.SetValue("")
	m.newKindMode = false
	m.clearHuhFields()
	m.fields = nil
	m.sourceID = ""
	m.sourceStatus = ""
	m.sourceProposedBy = ""
	m.sourceProposedAt = nil
	m.sourceTargetID = ""
	m.proposalID = ""
	m.delivery = entry.DeliveryInit
	m.focusIdx = 0
}

// ToEntry converts the form fields into an Entry.
func (m *Model) ToEntry() entry.Entry {
	e := entry.Entry{
		ID:         m.sourceID,
		Kind:       m.selectedKind(),
		Name:       strings.TrimSpace(m.huhVals.name),
		Group:      strings.TrimSpace(m.huhVals.group),
		Body:       strings.TrimSpace(m.huhVals.body),
		Delivery:   m.delivery,
		Status:     m.sourceStatus,
		ProposedBy: m.sourceProposedBy,
		ProposedAt: m.sourceProposedAt,
		TargetID:   m.sourceTargetID,
	}

	rawTags := strings.Split(m.huhVals.tags, ",")
	e.Tags = []string{}
	for _, t := range rawTags {
		t = strings.TrimSpace(t)
		if t != "" {
			e.Tags = append(e.Tags, t)
		}
	}

	e.Fields = make(map[string]string)
	for _, fp := range m.fields {
		k := strings.TrimSpace(fp.Key.Value())
		v := strings.TrimSpace(fp.Val.Value())
		if k != "" {
			e.Fields[k] = v
		}
	}

	return e
}

// SetDimensions sets the form dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	if m.huhForm != nil {
		m.rebuildHuhForm()
	}
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the form.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	// Kind dropdown result messages: apply the choice / close the panel.
	switch msg.(type) {
	case dropdownv2.ItemChosenMsg:
		return m.handleKindChosen(msg)
	case dropdownv2.ItemCanceledMsg:
		if m.kindDD != nil {
			m.kindDD, _ = m.kindDD.Update(msg)
		}
		m.syncKindFocus()
		return m, nil
	}

	// While the Kind panel is open, route all key/mouse events to it.
	if m.kindDD != nil && m.kindDD.Open() {
		switch msg.(type) {
		case tea.KeyPressMsg, tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseWheelMsg:
			m.kindDD, cmd = m.kindDD.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Save):
			e := m.ToEntry()
			pid := m.proposalID
			return m, func() tea.Msg { return SaveMsg{E: e, ProposalID: pid} }
		case key.Matches(msg, m.keys.Cancel):
			return m, func() tea.Msg { return CancelMsg{} }
		case key.Matches(msg, m.keys.Next):
			m.focusIdx = (m.focusIdx + 1) % m.fieldCount()
			m.applyFocus()
			if m.focusIdx == 1 && m.huhForm != nil {
				return m, m.huhForm.Init()
			}
		case key.Matches(msg, m.keys.Prev):
			m.focusIdx = (m.focusIdx - 1 + m.fieldCount()) % m.fieldCount()
			m.applyFocus()
			if m.focusIdx == 1 && m.huhForm != nil {
				return m, m.huhForm.Init()
			}
		case key.Matches(msg, m.keys.Submit):
			if m.focusIdx == m.focusIdxAddField() {
				m.addField()
			} else if m.focusIdx == m.focusIdxSave() {
				e := m.ToEntry()
				pid := m.proposalID
				return m, func() tea.Msg { return SaveMsg{E: e, ProposalID: pid} }
			} else if m.focusIdx == m.focusIdxCancel() {
				return m, func() tea.Msg { return CancelMsg{} }
			} else {
				m, cmd = m.updateFocusedField(msg)
			}
		default:
			m, cmd = m.updateFocusedField(msg)
		}

	case tea.MouseClickMsg:
		// A click on the Kind trigger opens the dropdown (even from new-kind
		// mode, so the user can switch back to an existing kind).
		if m.kindDD != nil && m.zoneManager.Get(zones.FormKindDD).InBounds(msg) {
			m.kindDD, cmd = m.kindDD.Update(msg)
			return m, cmd
		}
		if m.zoneManager.Get(zones.FormSave).InBounds(msg) {
			e := m.ToEntry()
			pid := m.proposalID
			return m, func() tea.Msg { return SaveMsg{E: e, ProposalID: pid} }
		} else if m.zoneManager.Get(zones.FormCancel).InBounds(msg) {
			return m, func() tea.Msg { return CancelMsg{} }
		} else if m.zoneManager.Get(zones.FormAddFld).InBounds(msg) {
			m.addField()
		} else {
			for i := range m.fields {
				if m.zoneManager.Get(zones.FormRemoveField(i)).InBounds(msg) {
					m.removeField(i)
					break
				}
			}
		}
	}

	return m, cmd
}

// updateFocusedField routes the key message to the focused input.
func (m Model) updateFocusedField(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focusIdx {
	case 0:
		// Index 0 is the Kind selector: the text input while typing a new
		// kind, otherwise the dropdown (which opens on Enter/Space and cycles
		// on ↑/↓ while focused).
		if m.newKindMode {
			m.kind, cmd = m.kind.Update(msg)
		} else if m.kindDD != nil {
			m.kindDD, cmd = m.kindDD.Update(msg)
		}
	case 1:
		if m.huhForm != nil {
			updated, huhCmd := m.huhForm.Update(msg)
			if f, ok := updated.(*huh.Form); ok {
				m.huhForm = f
			}
			cmd = huhCmd
		}
	default:
		fieldBase := 2
		for i := range m.fields {
			keyIdx := fieldBase + i*2
			valIdx := keyIdx + 1
			switch m.focusIdx {
			case keyIdx:
				m.fields[i].Key, cmd = m.fields[i].Key.Update(msg)
				return m, cmd
			case valIdx:
				m.fields[i].Val, cmd = m.fields[i].Val.Update(msg)
				return m, cmd
			}
		}
	}
	return m, cmd
}

// applyFocus sets focus on the correct input.
func (m *Model) applyFocus() {
	m.blurAll()
	switch m.focusIdx {
	case 0:
		// Only the text input takes keyboard focus directly; the dropdown's
		// focused state is set by syncKindFocus.
		if m.newKindMode {
			m.kind.Focus()
		}
		m.syncKindFocus()
	case 1:
		// huh manages its own field focus while this section is active.
	default:
		fieldBase := 2
		for i := range m.fields {
			keyIdx := fieldBase + i*2
			valIdx := keyIdx + 1
			switch m.focusIdx {
			case keyIdx:
				m.fields[i].Key.Focus()
				return
			case valIdx:
				m.fields[i].Val.Focus()
				return
			}
		}
	}
}

// blurAll blurs all inputs.
func (m *Model) blurAll() {
	if m.kindDD != nil {
		m.kindDD.SetFocused(false)
	}
	m.kind.Blur()
	for i := range m.fields {
		m.fields[i].Key.Blur()
		m.fields[i].Val.Blur()
	}
}

// addField appends a new empty field pair and focuses its key.
func (m *Model) addField() {
	kInput := textinput.New()
	kInput.Placeholder = "field name"
	kInput.CharLimit = 100

	vInput := textinput.New()
	vInput.Placeholder = "field value"
	vInput.CharLimit = 1000

	m.fields = append(m.fields, FieldPair{Key: kInput, Val: vInput})
	m.focusIdx = 2 + (len(m.fields)-1)*2
	m.applyFocus()
}

// removeField removes the field pair at index i.
func (m *Model) removeField(i int) {
	if i < 0 || i >= len(m.fields) {
		return
	}
	m.fields = append(m.fields[:i], m.fields[i+1:]...)
	total := m.fieldCount()
	if m.focusIdx >= total {
		m.focusIdx = total - 1
	}
	m.applyFocus()
}
