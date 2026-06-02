package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/naitv-mcp/internal/store"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/entries"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/form"
	"github.com/madicen/naitv-mcp/internal/tui/tabs/review"
)

const (
	tabEntries = 0
	tabReview  = 1
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
	store        *store.Store
	zoneManager  *zone.Manager
	activeTab    int
	entries      entries.Model
	review       review.Model
	form         form.Model
	pendingCount int
	width, height int
	statusMsg    string
	statusExpiry time.Time
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
		form:        form.NewModel(zm),
	}
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
			if !m.form.Visible() {
				return m, tea.Quit
			}
		}

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
		// Reload to get fresh kinds
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

	case pendingCountMsg:
		m.pendingCount = msg.count
		return m, nil
	}

	// If form is visible, route to form
	if m.form.Visible() {
		newForm, cmd := m.form.Update(msg)
		m.form = newForm
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Check tab bar zone clicks
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
	}

	// Route to active tab
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
	default:
		content = m.entries.View()
	}

	status := ""
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		status = "\n" + styleStatus.Render(m.statusMsg)
	}

	mainView := tabBar + "\n" + content + status

	if m.form.Visible() {
		formView := m.form.View()
		mainView = overlay.OverlayViewInCenter(mainView, formView, m.width, m.height)
	}

	return m.zoneManager.Scan(mainView)
}

// SetDimensions updates dimensions for all child models.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.entries.SetDimensions(w, h-1)
	m.review.SetDimensions(w, h-1)
	m.form.SetDimensions(w, h)
}

// renderTabBar renders the top tab bar.
func (m *Model) renderTabBar() string {
	entriesLabel := "Entries"
	reviewLabel := "Review"
	if m.pendingCount > 0 {
		reviewLabel = reviewLabel + " " + styleBadge.Render("("+itoa(m.pendingCount)+")")
	}

	var entriesTab, reviewTab string
	if m.activeTab == tabEntries {
		entriesTab = m.zoneManager.Mark("tab:entries", styleTabActive.Render(entriesLabel))
		reviewTab = m.zoneManager.Mark("tab:review", styleTabInactive.Render(reviewLabel))
	} else {
		entriesTab = m.zoneManager.Mark("tab:entries", styleTabInactive.Render(entriesLabel))
		reviewTab = m.zoneManager.Mark("tab:review", styleTabActive.Render(reviewLabel))
	}

	return styleTabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, entriesTab, reviewTab))
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
		m.form.Show()
	}

	if req.OpenEditForm {
		sel := m.entries.SelectedEntry()
		if sel != nil {
			m.form.Reset()
			m.form.SetMode(form.ModeEdit)
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
		// Update the entry first (store.Update the proposal fields)
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
