package plugins

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/madicen/naitv-mcp/internal/plugin"
	"github.com/madicen/naitv-mcp/internal/tui/layout"
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
	cursor        int
	width, height int
	viewport      viewport.Model

	// Text input for "install from custom source"
	inputActive bool
	input       textinput.Model

	// Status / loading
	status  string // flash message (success or error)
	loading bool   // true while an async op is in flight
}

// NewModel creates a new Plugins tab model.
func NewModel(zm *zone.Manager) Model {
	inp := textinput.New()
	inp.Placeholder = "plugin name, URL, or ./path/to/plugin.json"
	inp.CharLimit = 512
	inp.SetWidth(52)

	return Model{
		zoneManager:    zm,
		installedNames: make(map[string]bool),
		input:          inp,
	}
}

// Init returns the initial command (none — LoadPluginsCmd is called by root on tab switch).
func (m Model) Init() tea.Cmd { return nil }

// InputActive returns true when the text input for custom install is open.
// The root model uses this to suppress the global 'q' quit binding.
func (m Model) InputActive() bool { return m.inputActive }

// Update handles messages and returns (updated model, optional request, optional cmd).
func (m Model) Update(msg tea.Msg) (Model, *Request, tea.Cmd) {
	var req *Request

	switch msg := msg.(type) {

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
			m.cursor = 0
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
			switch msg.String() {
			case "esc":
				m.inputActive = false
				m.input.Blur()
				m.input.SetValue("")
				m.status = ""
			case "enter":
				source := strings.TrimSpace(m.input.Value())
				if source == "" {
					m.status = "Enter a plugin name, URL, or file path."
					return m, nil, nil
				}
				m.inputActive = false
				m.input.Blur()
				m.loading = true
				m.status = "Installing " + source + "…"
				req = &Request{Install: true, Source: source}
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, nil, cmd
			}
			return m, req, nil
		}

		switch msg.String() {
		case "j", "down":
			if m.cursor < m.listLen()-1 {
				m.cursor++
				m.updateViewport()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewport()
			}
		case "tab":
			// Switch between Installed / Browse; trigger registry fetch if switching to Browse.
			if m.mode == modeInstalled {
				if len(m.available) == 0 {
					m.loading = true
					m.status = "Fetching registry…"
					req = &Request{FetchRegistry: true}
				} else {
					m.mode = modeBrowse
					m.cursor = 0
					m.updateViewport()
				}
			} else {
				m.mode = modeInstalled
				m.cursor = 0
				m.updateViewport()
			}
		case "r":
			// Refresh registry (always re-fetches)
			m.loading = true
			m.status = "Fetching registry…"
			req = &Request{FetchRegistry: true}
		case "i":
			switch m.mode {
			case modeInstalled:
				// Open text input for custom source.
				m.inputActive = true
				m.input.Focus()
				m.status = ""
			case modeBrowse:
				// Install currently highlighted registry plugin.
				if sel := m.selectedAvailable(); sel != nil {
					if m.installedNames[sel.Name] {
						m.status = fmt.Sprintf("Plugin %q is already installed.", sel.Name)
					} else {
						m.loading = true
						m.status = "Installing " + sel.Name + "…"
						req = &Request{Install: true, Source: sel.Name}
					}
				}
			}
		case "u":
			if m.mode == modeInstalled {
				if sel := m.selectedInstalled(); sel != nil {
					m.loading = true
					m.status = "Uninstalling " + sel.Name + "…"
					req = &Request{Uninstall: true, Name: sel.Name}
				}
			}
		}
	}

	return m, req, nil
}

// View renders the plugins tab.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	left := m.renderList()
	right := m.renderDetail()
	bottom := m.renderBottom()

	divider := styleDiv.Render(strings.Repeat("│\n", m.contentH()))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
	return body + "\n" + bottom
}

// SetDimensions updates width/height and recalculates the detail viewport.
func (m *Model) SetDimensions(w, h int) {
	m.width = w
	m.height = h
	_, detailW := layout.SplitWidths(w)
	contentH := layout.ContentHeight(h, layout.PluginsFooterRows+2)
	vpW, vpH := layout.ViewportSize(detailW, contentH)
	m.viewport = viewport.New(viewport.WithWidth(vpW), viewport.WithHeight(vpH))
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
	if l := m.listLen(); l == 0 {
		m.cursor = 0
	} else if m.cursor >= l {
		m.cursor = l - 1
	}
}

func (m *Model) selectedInstalled() *entry.Entry {
	if len(m.installed) == 0 || m.cursor >= len(m.installed) {
		return nil
	}
	e := m.installed[m.cursor]
	return &e
}

func (m *Model) selectedAvailable() *plugin.RegistryEntry {
	if len(m.available) == 0 || m.cursor >= len(m.available) {
		return nil
	}
	e := m.available[m.cursor]
	return &e
}

func (m *Model) updateViewport() {
	m.viewport.SetContent(m.detailContent())
}

func (m *Model) listW() int {
	w, _ := layout.SplitWidths(m.width)
	return w
}
func (m *Model) detailW() int {
	_, w := layout.SplitWidths(m.width)
	return w
}
func (m *Model) contentH() int {
	return layout.ContentHeight(m.height, layout.PluginsFooterRows+2)
}

// ReloadInstalledMsg is emitted by the plugins tab to ask the root model to
// call LoadPluginsCmd(st). It is exported so the root can type-switch on it.
type ReloadInstalledMsg struct{}
