package plugins

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/tui/components/listpane"
	"github.com/madicen/naitv-mcp/internal/tui/keymap"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
	"github.com/madicen/naitv-mcp/internal/tui/theme"
	"github.com/madicen/naitv-mcp/internal/tui/zones"
	"github.com/madicen/naitv-mcp/pkg/entry"
)

// viewMode controls which list is shown in the left pane.
type viewMode int

const (
	modeInstalled viewMode = iota // showing installed plugins
	modeBrowse                    // showing registry plugins
)

// Request signals the root model to perform actions that require store access
// or cross-tab coordination.
type Request struct {
	// Install asks the root to call InstallPluginCmd(st, Source).
	Install bool
	Source  string

	// Uninstall asks the root to call UninstallPluginCmd(st, Name).
	Uninstall bool
	Name      string

	// FetchRegistry asks the root to call FetchRegistryCmd().
	FetchRegistry bool

	// RefreshPendingCount asks the root to reload the pending badge after a
	// successful install/uninstall.
	RefreshPendingCount bool
}

// Model is the state for the Plugins tab.
type Model struct {
	zoneManager *zone.Manager

	// Data
	installed []entry.Entry          // kind=plugin entries from store
	available []plugin.RegistryEntry // plugins listed in the registry

	// Installed-plugin name set — used in Browse mode to mark already-installed.
	installedNames map[string]bool

	// UI state
	mode          viewMode
	width, height int

	pane   listpane.Layout
	detail listpane.Detail
	sel    listpane.Selection

	// Text input for "install from custom source"
	inputActive bool
	input       textinput.Model

	// Status / loading
	status  string // flash message (success or error)
	loading bool   // true while an async op is in flight
	spin    spinner.Model
	keys    keymap.Plugins
}

// NewModel creates a new Plugins tab model.
func NewModel(zm *zone.Manager) Model {
	inp := textinput.New()
	inp.Placeholder = "plugin name, URL, or ./path/to/plugin.json"
	inp.CharLimit = 512
	inp.SetWidth(52)

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = theme.DimStyle

	return Model{
		zoneManager:    zm,
		installedNames: make(map[string]bool),
		input:          inp,
		detail:         listpane.NewDetail(),
		spin:           spin,
		keys:           keymap.DefaultPlugins,
	}
}

// Init returns the initial command (none — LoadPluginsCmd is called by root on tab switch).
func (m Model) Init() tea.Cmd { return nil }

func (m *Model) startLoading() tea.Cmd {
	m.loading = true
	return m.spin.Tick
}

// InputActive returns true when the text input for custom install is open.
// The root model uses this to suppress the global 'q' quit binding.
func (m Model) InputActive() bool { return m.inputActive }

// Update handles messages and returns (updated model, optional request, optional cmd).
func (m Model) Update(msg tea.Msg) (Model, *Request, tea.Cmd) {
	var req *Request
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.loading {
			m.spin, cmd = m.spin.Update(msg)
			return m, nil, cmd
		}
		return m, nil, nil

	// ── Async results ────────────────────────────────────────────────────────

	case PluginsLoadedMsg:
		if msg.Err != nil {
			m.status = "Error loading plugins: " + msg.Err.Error()
		} else {
			m.installed = msg.Entries
			m.installedNames = make(map[string]bool, len(msg.Entries))
			for _, e := range msg.Entries {
				m.installedNames[e.Name] = true
			}
		}
		m.loading = false
		m.clampCursor()
		m.updateViewport()
		return m, nil, nil

	case RegistryLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.status = "Registry fetch failed: " + msg.Err.Error()
		} else {
			m.available = msg.Registry.Plugins
			m.mode = modeBrowse
			m.sel.Index = 0
			m.status = fmt.Sprintf("Registry loaded — %d plugin(s) available", len(m.available))
		}
		m.updateViewport()
		return m, nil, nil

	case PluginInstalledMsg:
		m.loading = false
		m.inputActive = false
		m.input.Blur()
		m.input.SetValue("")
		if msg.Err != nil {
			m.status = "Install failed: " + msg.Err.Error()
		} else {
			r := msg.Result
			m.status = fmt.Sprintf("Plugin %q installed — %d entries pending approval", r.Manifest.Name, len(r.Proposed))
			req = &Request{RefreshPendingCount: true}
		}
		// Reload installed list regardless of success/failure.
		return m, req, func() tea.Msg { return ReloadInstalledMsg{} }

	case PluginUninstalledMsg:
		m.loading = false
		if msg.Err != nil {
			m.status = "Uninstall failed: " + msg.Err.Error()
		} else {
			m.status = fmt.Sprintf("Plugin %q removed (%d entries deleted)", msg.Result.Name, len(msg.Result.Removed))
			req = &Request{RefreshPendingCount: true}
		}
		return m, req, func() tea.Msg { return ReloadInstalledMsg{} }

	// ── Input mode (custom install source) ──────────────────────────────────

	case tea.KeyPressMsg:
		if m.inputActive {
			switch {
			case key.Matches(msg, m.keys.InputEsc):
				m.inputActive = false
				m.input.Blur()
				m.input.SetValue("")
				m.status = ""
			case key.Matches(msg, m.keys.InputEnter):
				source := strings.TrimSpace(m.input.Value())
				if source == "" {
					m.status = "Enter a plugin name, URL, or file path."
					return m, nil, nil
				}
				m.inputActive = false
				m.input.Blur()
				m.status = "Installing " + source + "…"
				req = &Request{Install: true, Source: source}
				return m, req, m.startLoading()
			default:
				m.input, cmd = m.input.Update(msg)
				return m, nil, cmd
			}
			return m, req, nil
		}

		switch {
		case key.Matches(msg, m.keys.Down):
			if m.sel.MoveDown(m.listLen()) {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Up):
			if m.sel.MoveUp() {
				m.updateViewport()
			}
		case key.Matches(msg, m.keys.Tab):
			if m.mode == modeInstalled {
				return m.activateMode(modeBrowse)
			}
			return m.activateMode(modeInstalled)
		case key.Matches(msg, m.keys.Refresh):
			return m.refreshRegistry()
		case key.Matches(msg, m.keys.Install):
			return m.doInstall()
		case key.Matches(msg, m.keys.Uninstall):
			return m.doUninstall()
		}

	case tea.MouseClickMsg:
		if m.inputActive {
			return m, nil, nil
		}
		switch {
		case m.zoneManager.Get(zones.PluginModeInstalled).InBounds(msg):
			return m.activateMode(modeInstalled)
		case m.zoneManager.Get(zones.PluginModeBrowse).InBounds(msg):
			return m.activateMode(modeBrowse)
		case m.zoneManager.Get(zones.PluginActInstall).InBounds(msg):
			return m.doInstall()
		case m.zoneManager.Get(zones.PluginActUninstall).InBounds(msg):
			return m.doUninstall()
		case m.zoneManager.Get(zones.PluginActTab).InBounds(msg):
			if m.mode == modeInstalled {
				return m.activateMode(modeBrowse)
			}
			return m.activateMode(modeInstalled)
		case m.zoneManager.Get(zones.PluginActRefresh).InBounds(msg):
			return m.refreshRegistry()
		default:
			for i := 0; i < m.listLen(); i++ {
				if m.zoneManager.Get(zones.PluginRow(i)).InBounds(msg) {
					m.sel.Index = i
					m.updateViewport()
					break
				}
			}
		}
		return m, nil, nil

	case tea.MouseWheelMsg:
		m.detail, cmd = m.detail.Update(msg)
		return m, nil, cmd
	}

	return m, req, cmd
}

// activateMode switches Installed/Browse. Browse with an empty registry fetches it.
func (m Model) activateMode(target viewMode) (Model, *Request, tea.Cmd) {
	if m.mode == target {
		return m, nil, nil
	}
	if target == modeBrowse && len(m.available) == 0 {
		return m.refreshRegistry()
	}
	m.mode = target
	m.sel.Index = 0
	m.updateViewport()
	return m, nil, nil
}

func (m Model) refreshRegistry() (Model, *Request, tea.Cmd) {
	m.status = "Fetching registry…"
	return m, &Request{FetchRegistry: true}, m.startLoading()
}

func (m Model) doInstall() (Model, *Request, tea.Cmd) {
	switch m.mode {
	case modeInstalled:
		m.inputActive = true
		m.input.Focus()
		m.status = ""
	case modeBrowse:
		if sel := m.selectedAvailable(); sel != nil {
			if m.installedNames[sel.Name] {
				m.status = fmt.Sprintf("Plugin %q is already installed.", sel.Name)
			} else {
				m.status = "Installing " + sel.Name + "…"
				return m, &Request{Install: true, Source: sel.Name}, m.startLoading()
			}
		}
	}
	return m, nil, nil
}

func (m Model) doUninstall() (Model, *Request, tea.Cmd) {
	if m.mode == modeInstalled {
		if sel := m.selectedInstalled(); sel != nil {
			m.status = "Uninstalling " + sel.Name + "…"
			return m, &Request{Uninstall: true, Name: sel.Name}, m.startLoading()
		}
	}
	return m, nil, nil
}

// View renders the plugins tab.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	left := m.renderList()
	right := m.renderDetail()
	bottom := m.renderBottom()

	divider := theme.PluginDivider.Render(strings.Repeat("│\n", m.contentH()))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
	return body + "\n" + bottom
}

// SetDimensions updates width/height and recalculates the detail viewport.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	m.pane = listpane.Compute(w, h, layout.PluginsFooterRows+2, 0)
	m.detail.Resize(m.pane)
	m.updateViewport()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (m *Model) listLen() int {
	if m.mode == modeInstalled {
		return len(m.installed)
	}
	return len(m.available)
}

func (m *Model) clampCursor() {
	m.sel.Clamp(m.listLen())
}

func (m *Model) selectedInstalled() *entry.Entry {
	if len(m.installed) == 0 || m.sel.Index >= len(m.installed) {
		return nil
	}
	e := m.installed[m.sel.Index]
	return &e
}

func (m *Model) selectedAvailable() *plugin.RegistryEntry {
	if len(m.available) == 0 || m.sel.Index >= len(m.available) {
		return nil
	}
	e := m.available[m.sel.Index]
	return &e
}

func (m *Model) updateViewport() {
	m.detail.SetContent(m.detailContent())
}

func (m *Model) listW() int   { return m.pane.ListW }
func (m *Model) detailW() int { return m.pane.DetailW }
func (m *Model) contentH() int {
	if m.height != 0 {
		return m.pane.ContentH
	}
	return layout.ContentHeight(m.height, layout.PluginsFooterRows+2)
}

// ReloadInstalledMsg is emitted by the plugins tab to ask the root model to
// call LoadPluginsCmd(st). It is exported so the root can type-switch on it.
type ReloadInstalledMsg struct{}
