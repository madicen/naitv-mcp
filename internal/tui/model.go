package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	dropdownv2 "github.com/madicen/bubble-dropdown/v2"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui/editor"
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
	undoHistoryID string
	editProposalID string
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
		m.loadStaleNudge(),
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
		if (msg.Text == "ctrl+c" || msg.Text == "q") && !m.form.Visible() && !m.tabs[m.activeTab].InputActive() {
			return m, tea.Quit
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

	case entries.EntriesLoadedMsg, entries.EntryDeletedMsg, entries.SearchResultsMsg, entries.HistoryLoadedMsg:
		tab, cmd := m.updateTab(tabEntries, msg)
		m.tabs[tabEntries] = tab
		cmds = append(cmds, cmd)
		if deleted, ok := msg.(entries.EntryDeletedMsg); ok {
			m.undoHistoryID = m.latestHistoryID(deleted.ID)
			m.setStatus(fmt.Sprintf("Deleted %q — u to undo", deleted.Name))
			kind := m.entriesTab().SelectedKind()
			if m.entriesTab().ShowArchived() {
				cmds = append(cmds, entries.LoadArchivedCmd(m.store, kind))
			} else {
				cmds = append(cmds, entries.LoadEntriesCmd(m.store, kind))
			}
		}
		return m, tea.Batch(cmds...)

	case editor.FinishedMsg:
		if m.editProposalID != "" {
			id := m.editProposalID
			m.editProposalID = ""
			if msg.Err != nil {
				m.setStatus("Editor failed: " + msg.Err.Error())
				return m, nil
			}
			e, err := m.store.Get(id)
			if err == nil {
				e.Body = msg.Body
				_, _ = m.store.Update(e)
				m.setStatus("Proposal body updated.")
			}
			return m, review.LoadProposalsCmd(m.store)
		}
		if m.form.Visible() {
			newForm, cmd := m.form.Update(msg)
			m.form = newForm
			return m, cmd
		}
		return m, nil

	case review.ProposalsLoadedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		return m, tea.Batch(cmd, review.LoadTargetsCmd(m.store, msg.Proposals))

	case review.TargetsLoadedMsg:
		tab, cmd := m.updateTab(tabReview, msg)
		m.tabs[tabReview] = tab
		return m, cmd

	case staleNudgeMsg:
		if msg.text != "" {
			m.setStatus(msg.text)
		}
		return m, nil

	case undoHintMsg:
		m.undoHistoryID = msg.historyID
		if msg.status != "" {
			m.setStatus(msg.status)
		}
		return m, nil

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

// EntriesSelectedName returns the name of the selected entry in the entries tab.
func (m *Model) EntriesSelectedName() string {
	e := m.entriesTab().SelectedEntry()
	if e == nil {
		return ""
	}
	return e.Name
}

// ZoneManager exposes the bubblezone manager for integration tests.
func (m *Model) ZoneManager() *zone.Manager { return m.zoneManager }

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
		cmds = append(cmds, m.formWindowSizeCmd())
	}
	if req.OpenEditForm {
		sel := et.SelectedEntry()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEdit)
			m.form.SetKinds(et.Kinds())
			m.form.PopulateFrom(*sel)
			m.form.Show()
			cmds = append(cmds, m.formWindowSizeCmd())
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
			cmds = append(cmds, m.captureUndoCmd(sel.ID, "Toggled delivery — u to undo"))
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
		if et.ShowArchived() {
			cmds = append(cmds, entries.LoadArchivedCmd(m.store, req.SwitchKind))
		} else {
			cmds = append(cmds, entries.LoadEntriesCmd(m.store, req.SwitchKind))
		}
	}
	if req.Search {
		cmds = append(cmds, entries.SearchCmd(m.store, et.SearchQuery()))
	}
	if req.Undo && m.undoHistoryID != "" {
		cmds = append(cmds, entries.UndoCmd(m.store, m.undoHistoryID, et.SelectedKind(), et.ShowArchived()))
		m.undoHistoryID = ""
		m.setStatus("Undone.")
	}
	if req.ShowHistory {
		sel := et.SelectedEntry()
		if sel != nil {
			cmds = append(cmds, entries.LoadHistoryCmd(m.store, sel.ID))
		}
	}
	if req.RestoreHistory {
		hid := et.SelectedHistoryID()
		if hid != "" {
			cmds = append(cmds, entries.RestoreVersionCmd(m.store, hid, et.SelectedKind(), et.ShowArchived()))
			m.setStatus("Restored historical version.")
		}
	}
	if req.ToggleArchive {
		kind := et.SelectedKind()
		if et.ShowArchived() {
			cmds = append(cmds, entries.LoadEntriesCmd(m.store, kind))
			m.setStatus("Showing active entries.")
		} else {
			cmds = append(cmds, entries.LoadArchivedCmd(m.store, kind))
			m.setStatus("Showing archived entries (v=restore, P=purge).")
		}
	}
	if req.RestoreEntry {
		sel := et.SelectedEntry()
		if sel != nil {
			cmds = append(cmds, entries.RestoreEntryCmd(m.store, sel.ID, et.SelectedKind()))
			m.setStatus("Entry restored.")
		}
	}
	if req.PurgeEntry {
		sel := et.SelectedEntry()
		if sel != nil {
			cmds = append(cmds, entries.PurgeEntryCmd(m.store, sel.ID, et.SelectedKind()))
			m.setStatus("Entry purged.")
		}
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
			cmds = append(cmds, m.formWindowSizeCmd())
		}
	}
	if req.EditBody {
		sel := rt.SelectedProposal()
		if sel != nil {
			m.editProposalID = sel.ID
			cmds = append(cmds, editor.OpenBodyCmd(sel.Body))
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

func (m *Model) formWindowSizeCmd() tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m *Model) loadStaleNudge() tea.Cmd {
	return func() tea.Msg {
		stale, err := m.store.StaleEntries(90, 3)
		if err != nil || len(stale) == 0 {
			return staleNudgeMsg{}
		}
		names := make([]string, 0, len(stale))
		for _, e := range stale {
			names = append(names, e.Name)
		}
		return staleNudgeMsg{text: "Stale entries (never accessed): " + strings.Join(names, ", ")}
	}
}

func (m *Model) captureUndoCmd(entryID, status string) tea.Cmd {
	return func() tea.Msg {
		return undoHintMsg{historyID: m.latestHistoryID(entryID), status: status}
	}
}

func (m *Model) latestHistoryID(entryID string) string {
	records, err := m.store.History(entryID)
	if err != nil || len(records) == 0 {
		return ""
	}
	return records[0].ID
}

type staleNudgeMsg struct{ text string }

type undoHintMsg struct {
	historyID string
	status    string
}

type pendingCountMsg struct{ count int }
