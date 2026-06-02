package entries

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// Request is returned from Update to signal actions the root model should handle.
type Request struct {
	OpenNewForm    bool
	OpenEditForm   bool
	DeleteSelected bool
	ConfirmDelete  bool
	ToggleDelivery bool
	SwitchToReview bool
	SwitchKind     string
	SwitchKindSet  bool // true even when SwitchKind == "" (all)
}

// Model holds the state for the entries tab.
type Model struct {
	zoneManager       *zone.Manager
	entries           []entry.Entry
	kinds             []string
	selectedKind      string
	selectedIdx       int
	width, height     int
	searchMode        bool
	searchInput       textinput.Model
	searchQuery       string
	viewport          viewport.Model
	showConfirmDelete bool
	deleteTargetID    string
}

// NewModel creates a new entries Model.
func NewModel(zm *zone.Manager) Model {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 200

	vp := viewport.New(0, 0)

	return Model{
		zoneManager: zm,
		searchInput: si,
		viewport:    vp,
	}
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated model, an optional request, and a command.
func (m Model) Update(msg tea.Msg) (Model, *Request, tea.Cmd) {
	var cmd tea.Cmd
	var req *Request

	switch msg := msg.(type) {
	case EntriesLoadedMsg:
		m.entries = msg.Entries
		m.kinds = msg.Kinds
		if m.selectedIdx >= len(m.entries) {
			m.selectedIdx = intMax(0, len(m.entries)-1)
		}
		m.updateViewport()
		return m, nil, nil

	case EntryDeletedMsg:
		newEntries := make([]entry.Entry, 0, len(m.entries))
		for _, e := range m.entries {
			if e.ID != msg.ID {
				newEntries = append(newEntries, e)
			}
		}
		m.entries = newEntries
		if m.selectedIdx >= len(m.entries) {
			m.selectedIdx = intMax(0, len(m.entries)-1)
		}
		m.showConfirmDelete = false
		m.deleteTargetID = ""
		m.updateViewport()
		return m, nil, nil

	case SearchResultsMsg:
		m.entries = msg.Entries
		m.searchQuery = msg.Query
		m.selectedIdx = 0
		m.updateViewport()
		return m, nil, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil, nil

	case tea.KeyMsg:
		if m.showConfirmDelete {
			switch msg.String() {
			case "y", "enter":
				req = &Request{ConfirmDelete: true}
				return m, req, nil
			case "n", "esc":
				m.showConfirmDelete = false
				m.deleteTargetID = ""
			}
			return m, nil, nil
		}

		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchQuery = ""
			case "enter":
				m.searchMode = false
				m.searchInput.Blur()
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
			}
			return m, nil, cmd
		}

		switch msg.String() {
		case "j", "down":
			if m.selectedIdx < len(m.entries)-1 {
				m.selectedIdx++
				m.updateViewport()
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.updateViewport()
			}
		case "n":
			req = &Request{OpenNewForm: true}
		case "e":
			if len(m.entries) > 0 {
				req = &Request{OpenEditForm: true}
			}
		case "d":
			if len(m.entries) > 0 {
				sel := m.SelectedEntry()
				if sel != nil {
					m.showConfirmDelete = true
					m.deleteTargetID = sel.ID
				}
			}
		case "i":
			if len(m.entries) > 0 {
				req = &Request{ToggleDelivery: true}
			}
		case "/":
			m.searchMode = true
			m.searchInput.Focus()
		case "R":
			req = &Request{SwitchToReview: true}
		case "tab":
			if len(m.kinds) > 0 {
				if m.selectedKind == "" {
					req = &Request{SwitchKind: m.kinds[0], SwitchKindSet: true}
				} else {
					found := false
					for i, k := range m.kinds {
						if k == m.selectedKind {
							if i+1 < len(m.kinds) {
								req = &Request{SwitchKind: m.kinds[i+1], SwitchKindSet: true}
							} else {
								req = &Request{SwitchKind: "", SwitchKindSet: true}
							}
							found = true
							break
						}
					}
					if !found {
						req = &Request{SwitchKind: "", SwitchKindSet: true}
					}
				}
			}
		}

	case tea.MouseMsg:
		m.viewport, cmd = m.viewport.Update(msg)

		if m.zoneManager.Get("action:new").InBounds(msg) {
			req = &Request{OpenNewForm: true}
		} else if m.zoneManager.Get("action:edit").InBounds(msg) {
			if len(m.entries) > 0 {
				req = &Request{OpenEditForm: true}
			}
		} else if m.zoneManager.Get("action:delete").InBounds(msg) {
			if len(m.entries) > 0 {
				sel := m.SelectedEntry()
				if sel != nil {
					m.showConfirmDelete = true
					m.deleteTargetID = sel.ID
				}
			}
		} else if m.zoneManager.Get("action:delivery").InBounds(msg) {
			if len(m.entries) > 0 {
				req = &Request{ToggleDelivery: true}
			}
		} else if m.zoneManager.Get("action:search").InBounds(msg) {
			m.searchMode = true
			m.searchInput.Focus()
		} else if m.zoneManager.Get("action:review").InBounds(msg) {
			req = &Request{SwitchToReview: true}
		} else {
			for i := range m.entries {
				if m.zoneManager.Get(entryRowZone(i)).InBounds(msg) {
					m.selectedIdx = i
					m.updateViewport()
					break
				}
			}
			if m.zoneManager.Get("kind:").InBounds(msg) {
				req = &Request{SwitchKind: "", SwitchKindSet: true}
			}
			for _, k := range m.kinds {
				if m.zoneManager.Get("kind:"+k).InBounds(msg) {
					req = &Request{SwitchKind: k, SwitchKindSet: true}
					break
				}
			}
		}
		return m, req, cmd
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, req, cmd
}

// SetDimensions updates the model's dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	listW := w * 35 / 100
	detailW := w - listW - 1
	contentH := h - 4
	if contentH < 1 {
		contentH = 1
	}
	// The detail pane has a rounded border, so the viewport's usable area is
	// 2 columns/rows smaller than the pane on each axis.
	vpW := detailW - 2
	if vpW < 1 {
		vpW = 1
	}
	vpH := contentH - 2
	if vpH < 1 {
		vpH = 1
	}
	m.viewport = viewport.New(vpW, vpH)
	m.updateViewport()
}

// SelectedEntry returns the currently selected entry or nil.
func (m *Model) SelectedEntry() *entry.Entry {
	if len(m.entries) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.entries) {
		return nil
	}
	e := m.entries[m.selectedIdx]
	return &e
}

// SetSelectedKind sets the active kind filter.
func (m *Model) SetSelectedKind(kind string) {
	m.selectedKind = kind
	m.selectedIdx = 0
}

// SelectedKind returns the currently active kind filter.
func (m *Model) SelectedKind() string {
	return m.selectedKind
}

// DeleteTargetID returns the ID of the entry pending deletion.
func (m *Model) DeleteTargetID() string {
	return m.deleteTargetID
}

// SearchQuery returns the current search query.
func (m *Model) SearchQuery() string {
	return m.searchInput.Value()
}

// updateViewport refreshes the viewport content.
func (m *Model) updateViewport() {
	sel := m.SelectedEntry()
	if sel == nil {
		m.viewport.SetContent("No entries.")
		return
	}
	m.viewport.SetContent(formatEntryDetail(*sel))
}

// formatEntryDetail formats an entry for the detail pane.
func formatEntryDetail(e entry.Entry) string {
	var sb strings.Builder
	sb.WriteString("Kind:   " + e.Kind + "\n")
	sb.WriteString("Name:   " + e.Name + "\n")
	sb.WriteString("ID:     " + e.ID + "\n")
	sb.WriteString("Deliver: " + deliveryLabel(e) + "\n")
	if len(e.Tags) > 0 {
		sb.WriteString("Tags:   " + strings.Join(e.Tags, ", ") + "\n")
	}
	if len(e.Fields) > 0 {
		sb.WriteString("\nFields:\n")
		for k, v := range e.Fields {
			sb.WriteString("  " + k + ": " + v + "\n")
		}
	}
	if e.Body != "" {
		sb.WriteString("\nBody:\n" + e.Body + "\n")
	}
	sb.WriteString("\nCreated: " + e.CreatedAt.Format("2006-01-02 15:04:05") + "\n")
	sb.WriteString("Updated: " + e.UpdatedAt.Format("2006-01-02 15:04:05") + "\n")
	return sb.String()
}

func entryRowZone(i int) string {
	return fmt.Sprintf("entry:%d", i)
}

// deliveryLabel describes an entry's delivery mode for the detail pane.
func deliveryLabel(e entry.Entry) string {
	if e.DeliveryOrDefault() == entry.DeliveryOnDemand {
		return "on-demand (agent must ask for it directly)"
	}
	return "init (included in initialization bundle)"
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
