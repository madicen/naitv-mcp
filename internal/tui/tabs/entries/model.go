package entries

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// itemKind distinguishes rows in the flat display list.
type itemKind int

const (
	itemKindHeader itemKind = iota // collapsible group header row
	itemKindEntry                  // regular entry row
)

// listItem is one row in the flattened entry list.
type listItem struct {
	kind      itemKind
	groupName string      // group this item belongs to (empty = General)
	count     int         // for header items: total entries in the group
	e         entry.Entry // for entry items
}

// Request is returned from Update to signal actions the root model should handle.
type Request struct {
	OpenNewForm    bool
	OpenEditForm   bool
	DeleteSelected bool
	ConfirmDelete  bool
	ToggleDelivery bool
	SwitchToReview bool
	CopyBody       bool
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

	// Group-collapse state.
	collapsed map[string]bool // group name → collapsed
	flatItems []listItem      // rebuilt by buildFlatItems after every data change

	// kindDD is the kind-filter dropdown (replaces the old pill row).
	kindDD *dropdownv2.Dropdown
	// contentTop is the absolute terminal row where the entries content begins
	// (the tab-bar height). Mouse events are translated by this offset before
	// reaching an open dropdown's geometric hit-test, since the dropdown panel
	// is composited in content-local coordinates inside View.
	contentTop int
}

// NewModel creates a new entries Model.
func NewModel(zm *zone.Manager) Model {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 200

	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))

	return Model{
		zoneManager: zm,
		searchInput: si,
		viewport:    vp,
		collapsed:   make(map[string]bool),
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

	// Dropdown result messages first: apply the choice, close the panel, and
	// (on a real selection) emit the kind-switch request.
	switch dm := msg.(type) {
	case dropdownv2.ItemChosenMsg:
		if m.kindDD != nil {
			m.kindDD, _ = m.kindDD.Update(msg)
		}
		req = &Request{SwitchKind: m.kindAtIndex(dm.Index), SwitchKindSet: true}
		return m, req, nil
	case dropdownv2.ItemCanceledMsg:
		if m.kindDD != nil {
			m.kindDD, _ = m.kindDD.Update(msg)
		}
		return m, nil, nil
	}

	// While the panel is open, route all key/mouse events to the dropdown
	// (it owns its own list navigation and geometric hit-testing). Mouse Y is
	// translated into content-local space first.
	if m.kindDD != nil && m.kindDD.Open() {
		switch mm := msg.(type) {
		case tea.KeyPressMsg:
			m.kindDD, cmd = m.kindDD.Update(mm)
			return m, nil, cmd
		case tea.MouseClickMsg:
			mm.Y -= m.contentTop
			m.kindDD, cmd = m.kindDD.Update(mm)
			return m, nil, cmd
		case tea.MouseMotionMsg:
			mm.Y -= m.contentTop
			m.kindDD, cmd = m.kindDD.Update(mm)
			return m, nil, cmd
		case tea.MouseWheelMsg:
			mm.Y -= m.contentTop
			m.kindDD, cmd = m.kindDD.Update(mm)
			return m, nil, cmd
		}
	}

	switch msg := msg.(type) {
	case EntriesLoadedMsg:
		m.entries = msg.Entries
		m.kinds = msg.Kinds
		m.buildFlatItems()
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
		m.showConfirmDelete = false
		m.deleteTargetID = ""
		m.buildFlatItems()
		m.updateViewport()
		return m, nil, nil

	case SearchResultsMsg:
		m.entries = msg.Entries
		m.searchQuery = msg.Query
		m.selectedIdx = 0
		m.buildFlatItems()
		m.updateViewport()
		return m, nil, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil, nil

	case tea.KeyPressMsg:
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
			if m.selectedIdx < len(m.flatItems)-1 {
				m.selectedIdx++
				m.updateViewport()
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.updateViewport()
			}
		case " ":
			// Space on a group header toggles its collapse state.
			if m.selectedIdx < len(m.flatItems) {
				item := m.flatItems[m.selectedIdx]
				if item.kind == itemKindHeader {
					m.collapsed[item.groupName] = !m.collapsed[item.groupName]
					m.buildFlatItems()
					m.updateViewport()
				}
			}
		case "n":
			req = &Request{OpenNewForm: true}
		case "e":
			if m.SelectedEntry() != nil {
				req = &Request{OpenEditForm: true}
			}
		case "d":
			sel := m.SelectedEntry()
			if sel != nil {
				m.showConfirmDelete = true
				m.deleteTargetID = sel.ID
			}
		case "i":
			if m.SelectedEntry() != nil {
				req = &Request{ToggleDelivery: true}
			}
		case "c":
			if m.SelectedEntry() != nil {
				req = &Request{CopyBody: true}
			}
		case "/":
			m.searchMode = true
			m.searchInput.Focus()
		case "R":
			req = &Request{SwitchToReview: true}
		case "tab":
			// Open the kind-filter dropdown. The dropdown opens on Enter when
			// focused, so synthesize one (it is kept focused; see
			// newKindDropdown). Return directly so the open command is not
			// clobbered by the trailing viewport update.
			if m.kindDD != nil {
				m.kindDD.SetFocused(true)
				m.kindDD, cmd = m.kindDD.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			}
			return m, nil, cmd
		}

	case tea.MouseClickMsg:
		// A click on the kind-filter trigger opens the dropdown (the panel is
		// then routed all events by the open-guard at the top of Update).
		if m.kindDD != nil && m.zoneManager.Get(kindDDZone).InBounds(msg) {
			m.kindDD, cmd = m.kindDD.Update(msg)
			return m, nil, cmd
		}

		if m.zoneManager.Get("action:new").InBounds(msg) {
			req = &Request{OpenNewForm: true}
		} else if m.zoneManager.Get("action:edit").InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{OpenEditForm: true}
			}
		} else if m.zoneManager.Get("action:delete").InBounds(msg) {
			sel := m.SelectedEntry()
			if sel != nil {
				m.showConfirmDelete = true
				m.deleteTargetID = sel.ID
			}
		} else if m.zoneManager.Get("action:delivery").InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{ToggleDelivery: true}
			}
		} else if m.zoneManager.Get("action:copy").InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{CopyBody: true}
			}
		} else if m.zoneManager.Get("action:search").InBounds(msg) {
			m.searchMode = true
			m.searchInput.Focus()
		} else if m.zoneManager.Get("action:review").InBounds(msg) {
			req = &Request{SwitchToReview: true}
		} else {
			for i, item := range m.flatItems {
				if m.zoneManager.Get(flatItemZone(i)).InBounds(msg) {
					if item.kind == itemKindHeader {
						// Clicking a header toggles its group.
						m.collapsed[item.groupName] = !m.collapsed[item.groupName]
						m.buildFlatItems()
					} else {
						m.selectedIdx = i
					}
					m.updateViewport()
					break
				}
			}
		}
		return m, req, cmd

	case tea.MouseWheelMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		return m, nil, cmd
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
	m.viewport = viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
	m.updateViewport()
}

// SelectedEntry returns the currently selected entry, or nil if the cursor is
// on a group header or the list is empty.
func (m *Model) SelectedEntry() *entry.Entry {
	if len(m.flatItems) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.flatItems) {
		return nil
	}
	item := m.flatItems[m.selectedIdx]
	if item.kind != itemKindEntry {
		return nil
	}
	e := item.e
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

// Kinds returns the distinct kinds currently known to the entries tab.
func (m *Model) Kinds() []string {
	return m.kinds
}

// SetContentTop records the absolute terminal row where the entries content
// begins (the tab-bar height). The root sets this so an open dropdown's mouse
// hit-test lines up with the composited panel.
func (m *Model) SetContentTop(top int) {
	if top < 0 {
		top = 0
	}
	m.contentTop = top
}

// DeleteTargetID returns the ID of the entry pending deletion.
func (m *Model) DeleteTargetID() string {
	return m.deleteTargetID
}

// SearchQuery returns the current search query.
func (m *Model) SearchQuery() string {
	return m.searchInput.Value()
}

// ── Grouping ──────────────────────────────────────────────────────────────────

// groupFor returns the display group for an entry. Precedence:
//  1. e.Group — explicitly set by the user or agent
//  2. plugin name from e.ProposedBy ("plugin:X" → "X")
//  3. "" — falls through to "General"
func groupFor(e entry.Entry) string {
	if e.Group != "" {
		return e.Group
	}
	if strings.HasPrefix(e.ProposedBy, "plugin:") {
		return strings.TrimPrefix(e.ProposedBy, "plugin:")
	}
	return ""
}

// buildFlatItems rebuilds the flat display list from m.entries and m.collapsed.
// Group headers are only emitted when at least one plugin group exists;
// otherwise the list stays flat (no headers) to preserve the simple experience
// when no plugins are installed.
func (m *Model) buildFlatItems() {
	type groupData struct {
		entries []entry.Entry
	}

	var order []string
	groups := map[string]*groupData{}

	for _, e := range m.entries {
		g := groupFor(e)
		if _, ok := groups[g]; !ok {
			order = append(order, g)
			groups[g] = &groupData{}
		}
		groups[g].entries = append(groups[g].entries, e)
	}

	// Sort: "" (General) first, then plugins alphabetically.
	sort.Slice(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a == "" {
			return true
		}
		if b == "" {
			return false
		}
		return a < b
	})

	// Only emit group headers when at least one plugin group is present.
	useHeaders := false
	for _, g := range order {
		if g != "" {
			useHeaders = true
			break
		}
	}

	m.flatItems = make([]listItem, 0, len(m.entries)+len(order))
	for _, gName := range order {
		gd := groups[gName]
		if useHeaders {
			m.flatItems = append(m.flatItems, listItem{
				kind:      itemKindHeader,
				groupName: gName,
				count:     len(gd.entries),
			})
		}
		if !m.collapsed[gName] {
			for _, e := range gd.entries {
				m.flatItems = append(m.flatItems, listItem{
					kind:      itemKindEntry,
					groupName: gName,
					e:         e,
				})
			}
		}
	}

	// Clamp the cursor.
	if l := len(m.flatItems); l == 0 {
		m.selectedIdx = 0
	} else if m.selectedIdx >= l {
		m.selectedIdx = l - 1
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

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

func flatItemZone(i int) string {
	return fmt.Sprintf("flat:%d", i)
}

// deliveryLabel describes an entry's delivery mode for the detail pane.
func deliveryLabel(e entry.Entry) string {
	if e.DeliveryOrDefault() == entry.DeliveryOnDemand {
		return "on-demand (agent must ask for it directly)"
	}
	return "init (included in initialization bundle)"
}
