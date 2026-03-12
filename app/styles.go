package app

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorMuted     = lipgloss.Color("#6B7280")
	colorError     = lipgloss.Color("#EF4444")
	colorSuccess   = lipgloss.Color("#10B981")
	colorWarning   = lipgloss.Color("#F59E0B")
	colorBg        = lipgloss.Color("#1F2937")
	colorHeaderBg  = lipgloss.Color("#374151")
	colorSelectBg  = lipgloss.Color("#4C1D95")

	// Tab styles
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorMuted)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Bold(true)

	// Table styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorHeaderBg).
			Padding(0, 1)

	typeRowStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Background(colorSelectBg).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	nullStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(0, 1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Stats view
	statLabelStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true).
			Width(14)

	statValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	statHeaderStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Underline(true).
			MarginBottom(1)

	// Section border
	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	// Error/info messages
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// Input prompt
	promptStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)
