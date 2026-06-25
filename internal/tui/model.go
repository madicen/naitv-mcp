package tui

import (
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	overlay "github.com/madicen/bubble-overlay"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/form"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/plugins"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
)

const (
	tabEntries = 0
	tabReview  = 1
	tabPlugins = 2
)

var (
	styleTabBar      = lipgloss.NewStyle().Padding(0, 1)
	styleTabActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Underline(true).Padding(0, 2)
	styleTabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 2)
	styleBadge       = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleStatus      = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Padding(0, 1)
)

// Model is the root bubbletea model.
type Model struct {
	store         *store.Store
	zoneManager   *zone.Manager
	activeTab     int
	entries       entries.Model
	review        review.Model
	pluginsTab    plugins.Model
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
		activeTab:   tabEntries,
		entries:     entries.NewModel(zm),
		review:      review.NewModel(zm),
		pluginsTab:  plugins.NewModel(zm),
		form:        form.NewModel(zm),
	}
	// The entries content is drawn two rows below the top (tab bar + blank
	// separator); tell the tab so its kind-filter dropdown's mouse hit-test
	// lines up with the panel.
	m.entries.SetContentTop(2)
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

	// Handle form messages first (they fire from form.Update via tea.Cmd)
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

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			if !m.form.Visible() && !m.pluginsTab.InputActive() {
				return m, tea.Quit
			}
		}

	// ── Dropdown results ─────────────────────────────────────────────────────
	// bubble-dropdown emits these as commands on a later tick; route them to
	// whichever surface owns an open dropdown (the form modal takes priority).

	case bubbledropdown.ItemChosenMsg, bubbledropdown.ItemCanceledMsg:
		if m.form.Visible() {
			newForm, cmd := m.form.Update(msg)
			m.form = newForm
			return m, cmd
		}
		if m.activeTab == tabEntries {
			newEntries, req, cmd := m.entries.Update(msg)
			m.entries = newEntries
			cmds = append(cmds, cmd)
			if req != nil {
				cmds = append(cmds, m.handleEntriesRequest(req))
			}
		}
		return m, tea.Batch(cmds...)

	// ── Entries tab messages ─────────────────────────────────────────────────

	case entries.EntriesLoadedMsg:
		newEntries, req, cmd := m.entries.Update(msg)
		m.entries = newEntries
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleEntriesRequest(req))
		}
		return m, tea.Batch(cmds...)

	case entries.EntryDeletedMsg:
		newEntries, req, cmd := m.entries.Update(msg)
		m.entries = newEntries
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleEntriesRequest(req))
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, m.entries.SelectedKind()))
		return m, tea.Batch(cmds...)

	case entries.SearchResultsMsg:
		newEntries, req, cmd := m.entries.Update(msg)
		m.entries = newEntries
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleEntriesRequest(req))
		}
		return m, tea.Batch(cmds...)

	// ── Review tab messages ──────────────────────────────────────────────────

	case review.ProposalsLoadedMsg:
		newReview, req, cmd := m.review.Update(msg)
		m.review = newReview
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleReviewRequest(req))
		}
		return m, tea.Batch(cmds...)

	case review.ProposalApprovedMsg:
		newReview, req, cmd := m.review.Update(msg)
		m.review = newReview
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleReviewRequest(req))
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, ""))
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("Proposal approved.")
		return m, tea.Batch(cmds...)

	case review.ProposalRejectedMsg:
		newReview, req, cmd := m.review.Update(msg)
		m.review = newReview
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleReviewRequest(req))
		}
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("Proposal rejected.")
		return m, tea.Batch(cmds...)

	case review.AllApprovedMsg:
		newReview, req, cmd := m.review.Update(msg)
		m.review = newReview
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleReviewRequest(req))
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, ""))
		cmds = append(cmds, m.loadPendingCount())
		m.setStatus("All proposals approved.")
		return m, tea.Batch(cmds...)

	// ── Plugins tab messages ─────────────────────────────────────────────────

	case plugins.PluginsLoadedMsg:
		newPlugins, req, cmd := m.pluginsTab.Update(msg)
		m.pluginsTab = newPlugins
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handlePluginsRequest(req))
		}
		return m, tea.Batch(cmds...)

	case plugins.RegistryLoadedMsg:
		newPlugins, req, cmd := m.pluginsTab.Update(msg)
		m.pluginsTab = newPlugins
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handlePluginsRequest(req))
		}
		return m, tea.Batch(cmds...)

	case plugins.PluginInstalledMsg:
		newPlugins, req, cmd := m.pluginsTab.Update(msg)
		m.pluginsTab = newPlugins
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handlePluginsRequest(req))
		}
		// After install, reload the installed list and pending count.
		cmds = append(cmds, plugins.LoadPluginsCmd(m.store))
		return m, tea.Batch(cmds...)

	case plugins.PluginUninstalledMsg:
		newPlugins, req, cmd := m.pluginsTab.Update(msg)
		m.pluginsTab = newPlugins
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handlePluginsRequest(req))
		}
		cmds = append(cmds, plugins.LoadPluginsCmd(m.store))
		return m, tea.Batch(cmds...)

	case plugins.ReloadInstalledMsg:
		cmds = append(cmds, plugins.LoadPluginsCmd(m.store))
		return m, tea.Batch(cmds...)

	// ── Misc ─────────────────────────────────────────────────────────────────

	case pendingCountMsg:
		m.pendingCount = msg.count
		return m, nil
	}

	// If form is visible, route to form.
	if m.form.Visible() {
		newForm, cmd := m.form.Update(msg)
		m.form = newForm
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Check tab bar zone clicks.
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		if m.zoneManager.Get("tab:entries").InBounds(mouseMsg) {
			m.activeTab = tabEntries
			return m, nil
		}
		if m.zoneManager.Get("tab:review").InBounds(mouseMsg) {
			m.activeTab = tabReview
			cmds = append(cmds, review.LoadProposalsCmd(m.store))
			return m, tea.Batch(cmds...)
		}
		if m.zoneManager.Get("tab:plugins").InBounds(mouseMsg) {
			m.activeTab = tabPlugins
			cmds = append(cmds, plugins.LoadPluginsCmd(m.store))
			return m, tea.Batch(cmds...)
		}
	}

	// Route key/mouse to active tab.
	switch m.activeTab {
	case tabEntries:
		newEntries, req, cmd := m.entries.Update(msg)
		m.entries = newEntries
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleEntriesRequest(req))
		}
	case tabReview:
		newReview, req, cmd := m.review.Update(msg)
		m.review = newReview
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handleReviewRequest(req))
		}
	case tabPlugins:
		newPlugins, req, cmd := m.pluginsTab.Update(msg)
		m.pluginsTab = newPlugins
		cmds = append(cmds, cmd)
		if req != nil {
			cmds = append(cmds, m.handlePluginsRequest(req))
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the full TUI.
func (m *Model) View() string {
	tabBar := m.renderTabBar()
	var content string

	switch m.activeTab {
	case tabEntries:
		content = m.entries.View()
	case tabReview:
		content = m.review.View()
	case tabPlugins:
		content = m.pluginsTab.View()
	default:
		content = m.entries.View()
	}

	status := ""
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		status = "\n" + styleStatus.Render(m.statusMsg)
	}

	// One blank line separates the tab bar from the active tab's content.
	mainView := tabBar + "\n\n" + content + status

	if m.form.Visible() {
		formView := m.form.View()
		mainView = overlay.OverlayViewInCenter(mainView, formView, m.width, m.height)
		// Composite the form's open Kind dropdown panel after centering, using
		// absolute bounds the form derives from the centered origin.
		mainView = m.form.ComposeDropdownOverlay(mainView, m.width, m.height)
	}

	return m.zoneManager.Scan(mainView)
}

// SetDimensions updates dimensions for all child models. Tabs receive h-2: one
// row for the tab bar and one for the blank separator line below it (see View).
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.entries.SetDimensions(w, h-2)
	m.review.SetDimensions(w, h-2)
	m.pluginsTab.SetDimensions(w, h-2)
	m.form.SetDimensions(w, h)
}

// renderTabBar renders the top tab bar.
func (m *Model) renderTabBar() string {
	entriesLabel := "Entries"
	reviewLabel := "Review"
	if m.pendingCount > 0 {
		reviewLabel = reviewLabel + " " + styleBadge.Render("("+itoa(m.pendingCount)+")")
	}
	pluginsLabel := "Plugins"

	render := func(label, zone string, active bool) string {
		style := styleTabInactive
		if active {
			style = styleTabActive
		}
		return m.zoneManager.Mark(zone, style.Render(label))
	}

	entriesTab := render(entriesLabel, "tab:entries", m.activeTab == tabEntries)
	reviewTab := render(reviewLabel, "tab:review", m.activeTab == tabReview)
	pluginsTab := render(pluginsLabel, "tab:plugins", m.activeTab == tabPlugins)

	return styleTabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, entriesTab, reviewTab, pluginsTab))
}

// handleEntriesRequest processes requests from the entries tab.
func (m *Model) handleEntriesRequest(req *entries.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	var cmds []tea.Cmd

	if req.OpenNewForm {
		m.form.Reset()
		m.form.SetMode(form.ModeCreate)
		m.form.SetKinds(m.entries.Kinds())
		m.form.Show()
	}
	if req.OpenEditForm {
		sel := m.entries.SelectedEntry()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEdit)
			m.form.SetKinds(m.entries.Kinds())
			m.form.PopulateFrom(*sel)
			m.form.Show()
		}
	}
	if req.ConfirmDelete {
		id := m.entries.DeleteTargetID()
		if id != "" {
			cmds = append(cmds, entries.DeleteEntryCmd(m.store, id))
		}
	}
	if req.ToggleDelivery {
		sel := m.entries.SelectedEntry()
		if sel != nil {
			cmds = append(cmds, entries.ToggleDeliveryCmd(m.store, sel.ID, m.entries.SelectedKind()))
		}
	}
	if req.CopyBody {
		sel := m.entries.SelectedEntry()
		switch {
		case sel == nil:
			m.setStatus("Nothing to copy")
		case sel.Body == "":
			m.setStatus("Entry has no body to copy")
		case clipboard.WriteAll(sel.Body) != nil:
			m.setStatus("Failed to copy to clipboard")
		default:
			m.setStatus("Copied body of \"" + sel.Name + "\" to clipboard")
		}
	}
	if req.SwitchToReview {
		m.activeTab = tabReview
		cmds = append(cmds, review.LoadProposalsCmd(m.store))
	}
	if req.SwitchKindSet {
		m.entries.SetSelectedKind(req.SwitchKind)
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, req.SwitchKind))
	}

	return tea.Batch(cmds...)
}

// handleReviewRequest processes requests from the review tab.
func (m *Model) handleReviewRequest(req *review.Request) tea.Cmd {
	if req == nil {
		return nil
	}
	var cmds []tea.Cmd

	if req.SwitchToEntries {
		m.activeTab = tabEntries
	}
	if req.ApproveSelected {
		id := m.review.SelectedID()
		if id != "" {
			cmds = append(cmds, review.ApproveCmd(m.store, id))
		}
	}
	if req.RejectSelected {
		id := m.review.SelectedID()
		if id != "" {
			cmds = append(cmds, review.RejectCmd(m.store, id))
		}
	}
	if req.EditSelected {
		sel := m.review.SelectedProposal()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEditProposal)
			m.form.SetKinds(m.entries.Kinds())
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

// handlePluginsRequest processes requests from the plugins tab.
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

// handleSave handles a SaveMsg from the form.
func (m *Model) handleSave(msg form.SaveMsg) tea.Cmd {
	var cmds []tea.Cmd
	switch m.form.GetMode() {
	case form.ModeCreate:
		e, err := m.store.Create(msg.E)
		if err == nil {
			m.setStatus("Entry created: " + e.Name)
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, m.entries.SelectedKind()))
	case form.ModeEdit:
		e, err := m.store.Update(msg.E)
		if err == nil {
			m.setStatus("Entry updated: " + e.Name)
		}
		cmds = append(cmds, entries.LoadEntriesCmd(m.store, m.entries.SelectedKind()))
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

// setStatus sets a status message that expires after 3 seconds.
func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

// loadPendingCount returns a command that loads the pending proposal count.
func (m *Model) loadPendingCount() tea.Cmd {
	return func() tea.Msg {
		count, _ := m.store.PendingCount()
		return pendingCountMsg{count: count}
	}
}

// pendingCountMsg carries the pending count.
type pendingCountMsg struct{ count int }

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
