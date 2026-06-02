package form

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Mode represents the form operation mode.
type Mode int

const (
	ModeCreate       Mode = iota
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
	name          textinput.Model
	tags          textinput.Model
	fields        []FieldPair
	body          textinput.Model
	sourceID      string
	proposalID    string
	delivery      entry.Delivery
	width, height int
	zoneManager   *zone.Manager
}

// fieldCount returns the total number of focusable items.
// Indices: 0=kind, 1=name, 2=tags, 3..3+len*2-1=fields, addField, body, save, cancel
func (m *Model) fieldCount() int {
	return 3 + len(m.fields)*2 + 1 + 1 + 2
}

func (m *Model) focusIdxAddField() int { return 3 + len(m.fields)*2 }
func (m *Model) focusIdxBody() int     { return m.focusIdxAddField() + 1 }
func (m *Model) focusIdxSave() int     { return m.focusIdxBody() + 1 }
func (m *Model) focusIdxCancel() int   { return m.focusIdxSave() + 1 }

// NewModel creates a new form Model.
func NewModel(zm *zone.Manager) Model {
	kind := textinput.New()
	kind.Placeholder = "kind (e.g. rule, fact, instruction)"
	kind.CharLimit = 100

	name := textinput.New()
	name.Placeholder = "name"
	name.CharLimit = 200

	tags := textinput.New()
	tags.Placeholder = "tags (comma-separated)"
	tags.CharLimit = 500

	body := textinput.New()
	body.Placeholder = "body / description"
	body.CharLimit = 5000

	return Model{
		kind:        kind,
		name:        name,
		tags:        tags,
		body:        body,
		zoneManager: zm,
	}
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
	m.kind.SetValue(e.Kind)
	m.name.SetValue(e.Name)
	m.tags.SetValue(strings.Join(e.Tags, ", "))
	m.body.SetValue(e.Body)
	m.sourceID = e.ID
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
	m.name.SetValue("")
	m.tags.SetValue("")
	m.body.SetValue("")
	m.fields = nil
	m.sourceID = ""
	m.proposalID = ""
	m.delivery = entry.DeliveryInit
	m.focusIdx = 0
}

// ToEntry converts the form fields into an Entry.
func (m *Model) ToEntry() entry.Entry {
	e := entry.Entry{
		ID:       m.sourceID,
		Kind:     strings.TrimSpace(m.kind.Value()),
		Name:     strings.TrimSpace(m.name.Value()),
		Body:     strings.TrimSpace(m.body.Value()),
		Delivery: m.delivery,
	}

	rawTags := strings.Split(m.tags.Value(), ",")
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
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages for the form.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			e := m.ToEntry()
			pid := m.proposalID
			return m, func() tea.Msg { return SaveMsg{E: e, ProposalID: pid} }
		case "esc":
			return m, func() tea.Msg { return CancelMsg{} }
		case "tab":
			m.focusIdx = (m.focusIdx + 1) % m.fieldCount()
			m.applyFocus()
		case "shift+tab":
			m.focusIdx = (m.focusIdx - 1 + m.fieldCount()) % m.fieldCount()
			m.applyFocus()
		case "enter":
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

	case tea.MouseMsg:
		if m.zoneManager.Get("form:save").InBounds(msg) {
			e := m.ToEntry()
			pid := m.proposalID
			return m, func() tea.Msg { return SaveMsg{E: e, ProposalID: pid} }
		} else if m.zoneManager.Get("form:cancel").InBounds(msg) {
			return m, func() tea.Msg { return CancelMsg{} }
		} else if m.zoneManager.Get("form:add-field").InBounds(msg) {
			m.addField()
		} else {
			for i := range m.fields {
				if m.zoneManager.Get(removeFieldZone(i)).InBounds(msg) {
					m.removeField(i)
					break
				}
			}
		}
	}

	return m, cmd
}

// updateFocusedField routes the key message to the focused input.
func (m Model) updateFocusedField(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focusIdx {
	case 0:
		m.kind, cmd = m.kind.Update(msg)
	case 1:
		m.name, cmd = m.name.Update(msg)
	case 2:
		m.tags, cmd = m.tags.Update(msg)
	default:
		fieldBase := 3
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
		if m.focusIdx == m.focusIdxBody() {
			m.body, cmd = m.body.Update(msg)
		}
	}
	return m, cmd
}

// applyFocus sets focus on the correct input.
func (m *Model) applyFocus() {
	m.blurAll()
	switch m.focusIdx {
	case 0:
		m.kind.Focus()
	case 1:
		m.name.Focus()
	case 2:
		m.tags.Focus()
	default:
		fieldBase := 3
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
		if m.focusIdx == m.focusIdxBody() {
			m.body.Focus()
		}
	}
}

// blurAll blurs all inputs.
func (m *Model) blurAll() {
	m.kind.Blur()
	m.name.Blur()
	m.tags.Blur()
	m.body.Blur()
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
	m.focusIdx = 3 + (len(m.fields)-1)*2
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

// removeFieldZone returns the zone ID for removing a field.
func removeFieldZone(i int) string {
	return fmt.Sprintf("form:remove-field:%d", i)
}
