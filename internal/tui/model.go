package tui

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui/tab"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/form"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/plugins"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
)

const (
	tabEntries = 0
	tabReview  = 1
	tabPlugins = 2
)

// Model is the root bubbletea model.
type Model struct {
	store         *store.Store
	zoneManager   *zone.Manager
	tabs          []tab.Tab
	activeTab     int
	form          form.Model
	pendingCount  int
	width, height int
	statusMsg     string
	statusExpiry  time.Time
}

// New creates a new root Model.
func New(st *store.Store) *Model {
	zm := zone.New()

	m := &Model{
		store:       st,
		zoneManager: zm,
		tabs: []tab.Tab{
			entries.NewTab(zm),
			review.NewTab(zm),
			plugins.NewTab(zm),
		},
		activeTab: tabEntries,
		form:      form.NewModel(zm),
	}
	// The entries content is drawn two rows below the top (tab bar + blank
	// separator); tell the tab so its kind-filter dropdown's mouse hit-test
	// lines up with the panel.
	m.tabs[tabEntries].SetContentTop(2)
	return m
}

// Init returns the initial commands.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		entries.LoadEntriesCmd(m.store, ""),
		m.loadPendingCount(),
	)
}

// Update handles all messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case form.SaveMsg:
		cmds = append(cmds, m.handleSave(msg))
		m.form.Hide()
		return m, tea.Batch(cmds...)

	case form.CancelMsg:
		m.form.Hide()
		return m, nil

	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if !m.form.Visible() && !m.tabs[m.activeTab].InputActive() {
				return m, tea.Quit
			}
		}

	case dropdownv2.ItemChosenMsg, dropdownv2.ItemCanceledMsg:
		if m.form.Visible() {
			newForm, cmd := m.form.Update(msg)
			m.form = newForm
			return m, cmd
		}
		tab, cmd := m.tabs[m.activeTab].Update(msg)
		m.tabs[m.activeTab] = tab
		return m, cmd

	case entries.RequestMsg:
		return m, m.handleEntriesRequest(&msg.Req)

	case review.RequestMsg:
		return m, m.handleReviewRequest(&msg.Req)

	case plugins.RequestMsg:
		return m, m.handlePluginsRequest(&msg.Req)

	case entries.EntriesLoadedMsg, entries.EntryDeletedMsg, entries.SearchResultsMsg:
		tab, cmd := m.updateTab(tabEntries, msg)
		m.tabs[tabEntries] = tab
		cmds = append(cmds, cmd)
		if _, ok := msg.(entries.EntryDeletedMsg); ok {
			cmds = append(cmds, entries.LoadEntriesCmd(m.store, m.entriesTab().SelectedKind()))
		}
		return m, tea.Batch(cmds...)

	case review.ProposalsLoadedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		return m, cmd

	case review.ProposalApprovedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		cmds = append(cmds, cmd)
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, ""))
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("Proposal approved.")
		return m, tea.Batch(cmds...)

	case review.ProposalRejectedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		cmds = append(cmds, cmd)
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("Proposal rejected.")
		return m, tea.Batch(cmds...)

	case review.AllApprovedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		cmds = append(cmds, cmd)
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, ""))
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("All proposals approved.")
		return m, tea.Batch(cmds...)

	case plugins.PluginsLoadedMsg, plugins.RegistryLoadedMsg:
		tab, cmd := m.updateTab(tabPlugins, msg)
		m.tabs[tabPlugins] = tab
		return m, cmd

	case plugins.PluginInstalledMsg, plugins.PluginUninstalledMsg:
		tab, cmd := m.updateTab(tabPlugins, msg)
		m.tabs[tabPlugins] = tab
		cmds = append(cmds, cmd)
		cmds = append(cmds, plugins.LoadPluginsCmd(m.store))
		return m, tea.Batch(cmds...)

	case plugins.ReloadInstalledMsg:
		return m, plugins.LoadPluginsCmd(m.store)

	case pendingCountMsg:
		m.pendingCount = msg.count
		return m, nil
	}

	if m.form.Visible() {
		newForm, cmd := m.form.Update(msg)
		m.form = newForm
		return m, cmd
	}

	if clickMsg, ok := msg.(tea.MouseClickMsg); ok {
		if m.zoneManager.Get(zones.TabEntries).InBounds(clickMsg) {
			m.activeTab = tabEntries
			return m, nil
		}
		if m.zoneManager.Get(zones.TabReview).InBounds(clickMsg) {
			m.activeTab = tabReview
			return m, review.LoadProposalsCmd(m.store)
		}
		if m.zoneManager.Get(zones.TabPlugins).InBounds(clickMsg) {
			m.activeTab = tabPlugins
			return m, plugins.LoadPluginsCmd(m.store)
		}
	}

	tab, cmd := m.tabs[m.activeTab].Update(msg)
	m.tabs[m.activeTab] = tab
	return m, cmd
}

func (m *Model) updateTab(i int, msg tea.Msg) (tab.Tab, tea.Cmd) {
	tab, cmd := m.tabs[i].Update(msg)
	return tab, cmd
}

func (m *Model) entriesTab() *entries.Tab {
	return m.tabs[tabEntries].(*entries.Tab)
}

func (m *Model) reviewTab() *review.Tab {
	return m.tabs[tabReview].(*review.Tab)
}

// View renders the full TUI.
func (m *Model) View() tea.View {
	tabBar := m.renderTabBar()
	content := m.tabs[m.activeTab].View()

	status := ""
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		status = "\n" + theme.StatusStyle.Render(m.statusMsg)
	}

	mainView := tabBar + "\n\n" + content + status

	if m.form.Visible() {
		formView := m.form.View()
		mainView = overlay.OverlayViewInCenter(mainView, formView, m.width, m.height)
		mainView = m.form.ComposeDropdownOverlay(mainView, m.width, m.height)
	}

	v := tea.NewView(m.zoneManager.Scan(mainView))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// SetDimensions updates dimensions for all child models. Tabs receive h-2: one
// row for the tab bar and one for the blank separator line below it (see View).
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	tabH := h - 2
	for i := range m.tabs {
		m.tabs[i].SetDimensions(w, tabH)
	}
	m.form.SetDimensions(w, h)
}

// renderTabBar renders the top tab bar.
func (m *Model) renderTabBar() string {
	entriesLabel := "Entries"
	reviewLabel := "Review"
	if m.pendingCount > 0 {
		reviewLabel = reviewLabel + " " + theme.BadgeStyle.Render("("+strconv.Itoa(m.pendingCount)+")")
	}
	pluginsLabel := "Plugins"

	render := func(label, zoneID string, active bool) string {
		style := theme.TabInactive
		if active {
			style = theme.TabActive
		}
		return m.zoneManager.Mark(zoneID, style.Render(label))
	}

	entriesTabLabel := render(entriesLabel, zones.TabEntries, m.activeTab == tabEntries)
	reviewTabLabel := render(reviewLabel, zones.TabReview, m.activeTab == tabReview)
	pluginsTabLabel := render(pluginsLabel, zones.TabPlugins, m.activeTab == tabPlugins)

	return theme.TabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, entriesTabLabel, reviewTabLabel, pluginsTabLabel))
}

func (m *Model) handleEntriesRequest(req *entries.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	var cmds []tea.Cmd
	et := m.entriesTab()

	if req.OpenNewForm {
		m.form.Reset()
		m.form.SetMode(form.ModeCreate)
		m.form.SetKinds(et.Kinds())
		m.form.Show()
	}
	if req.OpenEditForm {
		sel := et.SelectedEntry()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEdit)
			m.form.SetKinds(et.Kinds())
			m.form.PopulateFrom(*sel)
			m.form.Show()
		}
	}
	if req.ConfirmDelete {
		id := et.DeleteTargetID()
		if id != "" {
			cmds = append(cmds, entries.DeleteEntryCmd(m.store, id))
		}
	}
	if req.ToggleDelivery {
		sel := et.SelectedEntry()
		if sel != nil {
			cmds = append(cmds, entries.ToggleDeliveryCmd(m.store, sel.ID, et.SelectedKind()))
		}
	}
	if req.CopyBody {
		sel := et.SelectedEntry()
		switch {
		case sel == nil:
			m.setStatus("Nothing to copy")
		case sel.Body == "":
			m.setStatus("Entry has no body to copy")
		default:
			m.setStatus("Copied body of \"" + sel.Name + "\" to clipboard")
			cmds = append(cmds, tea.SetClipboard(sel.Body))
		}
	}
	if req.SwitchToReview {
		m.activeTab = tabReview
		cmds = append(cmds, review.LoadProposalsCmd(m.store))
	}
	if req.SwitchKindSet {
		m.entriesTab().SetSelectedKind(req.SwitchKind)
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, req.SwitchKind))
	}

	return tea.Batch(cmds...)
}

func (m *Model) handleReviewRequest(req *review.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	var cmds []tea.Cmd
	rt := m.reviewTab()
	et := m.entriesTab()

	if req.SwitchToEntries {
		m.activeTab = tabEntries
	}
	if req.ApproveSelected {
		id := rt.SelectedID()
		if id != "" {
			cmds = append(cmds, review.ApproveCmd(m.store, id))
		}
	}
	if req.RejectSelected {
		id := rt.SelectedID()
		if id != "" {
			cmds = append(cmds, review.RejectCmd(m.store, id))
		}
	}
	if req.EditSelected {
		sel := rt.SelectedProposal()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEditProposal)
			m.form.SetKinds(et.Kinds())
			m.form.PopulateFrom(*sel)
			m.form.SetProposalID(sel.ID)
			m.form.Show()
		}
	}
	if req.ApproveAll {
		cmds = append(cmds, review.ApproveAllCmd(m.store))
	}

	return tea.Batch(cmds...)
}

func (m *Model) handlePluginsRequest(req *plugins.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	var cmds []tea.Cmd

	if req.Install {
		cmds = append(cmds, plugins.InstallPluginCmd(m.store, req.Source))
	}
	if req.Uninstall {
		cmds = append(cmds, plugins.UninstallPluginCmd(m.store, req.Name))
	}
	if req.FetchRegistry {
		cmds = append(cmds, plugins.FetchRegistryCmd())
	}
	if req.RefreshPendingCount {
		cmds = append(cmds, m.loadPendingCount())
	}

	return tea.Batch(cmds...)
}

func (m *Model) handleSave(msg form.SaveMsg) tea.Cmd {
	var cmds []tea.Cmd
	et := m.entriesTab()
	switch m.form.GetMode() {
	case form.ModeCreate:
		e, err := m.store.Create(msg.E)
		if err == nil {
			m.setStatus("Entry created: " + e.Name)
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, et.SelectedKind()))
	case form.ModeEdit:
		e, err := m.store.Update(msg.E)
		if err == nil {
			m.setStatus("Entry updated: " + e.Name)
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, et.SelectedKind()))
	case form.ModeEditProposal:
		_, err := m.store.Update(msg.E)
		if err == nil && msg.ProposalID != "" {
			_, _ = m.store.Approve(msg.ProposalID)
			m.setStatus("Proposal approved after edit.")
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, ""))
		cmds = append(cmds, review.LoadProposalsCmd(m.store))
		cmds = append(cmds, m.loadPendingCount())
	}
	return tea.Batch(cmds...)
}

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

func (m *Model) loadPendingCount() tea.Cmd {
	return func() tea.Msg {
		count, _ := m.store.PendingCount()
		return pendingCountMsg{count: count}
	}
}

type pendingCountMsg struct{ count int }
