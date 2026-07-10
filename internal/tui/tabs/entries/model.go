package entries

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
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
	width, height     int
	searchMode        bool
	searchInput       textinput.Model
	searchQuery       string
	showConfirmDelete bool
	deleteTargetID    string

	pane   listpane.Layout
	detail listpane.Detail
	sel    listpane.Selection

	// Group-collapse state.
	collapsed map[string]bool // group name → collapsed
	flatItems []listItem      // rebuilt by buildFlatItems after every data change

	// kindDD is the kind-filter dropdown (replaces the old pill row).
	kindDD *dropdownv2.Dropdown
	keys   keymap.Entries
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

	return Model{
		zoneManager: zm,
		searchInput: si,
		detail:      listpane.NewDetail(),
		collapsed:   make(map[string]bool),
		keys:        keymap.DefaultEntries,
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
		m.sel.Index = 0
		m.buildFlatItems()
		m.updateViewport()
		return m, nil, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil, nil

	case tea.KeyPressMsg:
		if m.showConfirmDelete {
			switch {
			case key.Matches(msg, m.keys.ConfirmYes):
				req = &Request{ConfirmDelete: true}
				return m, req, nil
			case key.Matches(msg, m.keys.ConfirmNo):
				m.showConfirmDelete = false
				m.deleteTargetID = ""
			}
			return m, nil, nil
		}

		if m.searchMode {
			switch {
			case key.Matches(msg, m.keys.SearchEsc):
				m.searchMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchQuery = ""
			case msg.String() == "enter":
				m.searchMode = false
				m.searchInput.Blur()
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
			}
			return m, nil, cmd
		}

		switch {
		case key.Matches(msg, m.keys.Down):
			if m.sel.MoveDown(len(m.flatItems)) {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Up):
			if m.sel.MoveUp() {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Space):
			// Space on a group header toggles its collapse state.
			if m.sel.Index < len(m.flatItems) {
				item := m.flatItems[m.sel.Index]
				if item.kind == itemKindHeader {
					m.collapsed[item.groupName] = !m.collapsed[item.groupName]
					m.buildFlatItems()
					m.updateViewport()
				}
			}
		case key.Matches(msg, m.keys.New):
			req = &Request{OpenNewForm: true}
		case key.Matches(msg, m.keys.Edit):
			if m.SelectedEntry() != nil {
				req = &Request{OpenEditForm: true}
			}
		case key.Matches(msg, m.keys.Delete):
			sel := m.SelectedEntry()
			if sel != nil {
				m.showConfirmDelete = true
				m.deleteTargetID = sel.ID
			}
		case key.Matches(msg, m.keys.Delivery):
			if m.SelectedEntry() != nil {
				req = &Request{ToggleDelivery: true}
			}
		case key.Matches(msg, m.keys.Copy):
			if m.SelectedEntry() != nil {
				req = &Request{CopyBody: true}
			}
		case key.Matches(msg, m.keys.Search):
			m.searchMode = true
			m.searchInput.Focus()
		case key.Matches(msg, m.keys.Review):
			req = &Request{SwitchToReview: true}
		case key.Matches(msg, m.keys.Tab):
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
		if m.kindDD != nil && m.zoneManager.Get(zones.EntriesKindDD).InBounds(msg) {
			m.kindDD, cmd = m.kindDD.Update(msg)
			return m, nil, cmd
		}

		if m.zoneManager.Get(zones.EntriesNew).InBounds(msg) {
			req = &Request{OpenNewForm: true}
		} else if m.zoneManager.Get(zones.EntriesEdit).InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{OpenEditForm: true}
			}
		} else if m.zoneManager.Get(zones.EntriesDelete).InBounds(msg) {
			sel := m.SelectedEntry()
			if sel != nil {
				m.showConfirmDelete = true
				m.deleteTargetID = sel.ID
			}
		} else if m.zoneManager.Get(zones.EntriesDelivery).InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{ToggleDelivery: true}
			}
		} else if m.zoneManager.Get(zones.EntriesCopy).InBounds(msg) {
			if m.SelectedEntry() != nil {
				req = &Request{CopyBody: true}
			}
		} else if m.zoneManager.Get(zones.EntriesSearch).InBounds(msg) {
			m.searchMode = true
			m.searchInput.Focus()
		} else if m.zoneManager.Get(zones.EntriesReview).InBounds(msg) {
			req = &Request{SwitchToReview: true}
		} else {
			for i, item := range m.flatItems {
				if m.zoneManager.Get(zones.EntriesRow(i)).InBounds(msg) {
					if item.kind == itemKindHeader {
						// Clicking a header toggles its group.
						m.collapsed[item.groupName] = !m.collapsed[item.groupName]
						m.buildFlatItems()
					} else {
						m.sel.Index = i
					}
					m.updateViewport()
					break
				}
			}
		}
		return m, req, cmd

	case tea.MouseWheelMsg:
		m.detail, cmd = m.detail.Update(msg)
		return m, nil, cmd
	}

	m.detail, cmd = m.detail.Update(msg)
	return m, req, cmd
}

// SetDimensions updates the model's dimensions.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.pane = listpane.Compute(w, h, layout.EntriesFooterRows+2, 0)
	m.detail.Resize(m.pane)
	m.updateViewport()
}

// SelectedEntry returns the currently selected entry, or nil if the cursor is
// on a group header or the list is empty.
func (m *Model) SelectedEntry() *entry.Entry {
	if len(m.flatItems) == 0 || m.sel.Index < 0 || m.sel.Index >= len(m.flatItems) {
		return nil
	}
	item := m.flatItems[m.sel.Index]
	if item.kind != itemKindEntry {
		return nil
	}
	e := item.e
	return &e
}

// SetSelectedKind sets the active kind filter.
func (m *Model) SetSelectedKind(kind string) {
	m.selectedKind = kind
	m.sel.Index = 0
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
		m.sel.Index = 0
	} else {
		m.sel.Clamp(l)
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// updateViewport refreshes the viewport content.
func (m *Model) updateViewport() {
	sel := m.SelectedEntry()
	if sel == nil {
		m.detail.SetContent("No entries.")
		return
	}
	m.detail.SetContent(formatEntryDetail(*sel))
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

// deliveryLabel describes an entry's delivery mode for the detail pane.
func deliveryLabel(e entry.Entry) string {
	if e.DeliveryOrDefault() == entry.DeliveryOnDemand {
		return "on-demand (agent must ask for it directly)"
	}
	return "init (included in initialization bundle)"
}
