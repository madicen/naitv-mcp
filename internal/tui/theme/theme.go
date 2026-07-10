// Package theme holds shared visual constants and styles for the TUI.
package theme

import "charm.land/lipgloss/v2"

const (
	Accent   = "205"
	Dim      = "240"
	Info     = "39"
	Text     = "252"
	Status   = "220"
	Badge    = "196"
	GroupSel = "39"
	Init     = "42"
	ExecTool = "214"
	BadgeNew = "46"
	BadgeUpd = "220"
	Installed = "34"
)

var (
	Selected       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Accent))
	DimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(Dim))
	TextStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(Text))
	Pane           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(Dim)).Padding(0, 1)
	Title          = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Accent)).Padding(0, 1)
	StatusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(Status)).Padding(0, 1)
	BadgeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(Badge)).Bold(true)
	TabActive      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Accent)).Underline(true).Padding(0, 2)
	TabInactive    = lipgloss.NewStyle().Foreground(lipgloss.Color(Dim)).Padding(0, 2)
	TabBar         = lipgloss.NewStyle().Padding(0, 1)
	KeyHint        = lipgloss.NewStyle().Foreground(lipgloss.Color(Accent)).Bold(true)
	Hint           = lipgloss.NewStyle().Foreground(lipgloss.Color(Dim))
	FormFocused    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(Accent)).Padding(0, 1)
	FormPanel      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(Accent)).Padding(1, 2)
	FormLabel      = lipgloss.NewStyle().Foreground(lipgloss.Color(Text)).Width(10)
	FormInput      = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(Dim)).Padding(0, 1)
	FormBtn        = lipgloss.NewStyle().Foreground(lipgloss.Color(Info)).Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(Info))
	FormBtnActive  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color(Info)).Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(Info))
	FormRemoveBtn  = lipgloss.NewStyle().Foreground(lipgloss.Color(Badge)).Padding(0, 1)
	FormDimLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color(Dim))
	GroupHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Info))
	GroupHeaderSel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Accent))
	GroupCount     = DimStyle
	ActionBtn      = lipgloss.NewStyle().Foreground(lipgloss.Color(Info)).Padding(0, 1)
	Confirm        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Badge))
	SearchBar      = lipgloss.NewStyle().Foreground(lipgloss.Color(Status)).Padding(0, 1)
	InitGlyph      = lipgloss.NewStyle().Foreground(lipgloss.Color(Init))
	OnDemandGlyph  = DimStyle
	ExecToolGlyph  = lipgloss.NewStyle().Foreground(lipgloss.Color(ExecTool))
	BadgeNewStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(BadgeNew)).Padding(0, 1)
	BadgeUpdStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(BadgeUpd)).Padding(0, 1)
	PluginMode     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Info)).Padding(0, 1)
	PluginModeInact = lipgloss.NewStyle().Foreground(lipgloss.Color(Dim)).Padding(0, 1)
	PluginDivider  = DimStyle
	PluginDetail   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(Dim))
	PluginList     = lipgloss.NewStyle().Padding(0, 1)
	PluginInstalled = lipgloss.NewStyle().Foreground(lipgloss.Color(Installed)).Bold(true)
)
