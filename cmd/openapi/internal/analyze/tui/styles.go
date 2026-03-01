package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorGreen       = "#10B981"
	colorBlue        = "#3B82F6"
	colorYellow      = "#F59E0B"
	colorRed         = "#EF4444"
	colorPurple      = "#8B5CF6"
	colorGray        = "#6B7280"
	colorThemePurple = "#7C3AED"
	colorBackground  = "#374151"
	colorDetailGray  = "#9CA3AF"
	colorFooterText  = "#E5E7EB"
	colorWhite       = "#FFFFFF"
	colorCyan        = "#06B6D4"
	colorOrange      = "#F97316"
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorThemePurple))

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDetailGray))

	// Tier badge styles
	GreenBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen)).
			Bold(true)

	YellowBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow)).
			Bold(true)

	RedBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed)).
			Bold(true)

	// Stat styles
	StatLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDetailGray))

	StatValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite)).
			Bold(true)

	StatHighlight = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan)).
			Bold(true)

	StatWarning = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorOrange)).
			Bold(true)

	// Tab styles
	ActiveTab = lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color(colorThemePurple)).
			Foreground(lipgloss.Color(colorWhite)).
			Bold(true)

	InactiveTab = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(colorGray))

	// List styles
	SelectedRow = lipgloss.NewStyle().
			Background(lipgloss.Color(colorBackground))

	NormalRow = lipgloss.NewStyle()

	// Detail section
	DetailStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(colorDetailGray))

	// Suggestion styles
	SuggestionStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(colorCyan))

	// Cycle edge styles
	RequiredEdge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed)).
			Bold(true)

	ArrayEdge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue))

	// Footer
	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(colorGray)).
			Foreground(lipgloss.Color(colorFooterText)).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	HelpTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite))

	HelpModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorThemePurple)).
			Padding(1, 2).
			Width(50)

	HelpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorThemePurple)).
			Align(lipgloss.Center).
			Width(46)

	CardTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorThemePurple)).
			Bold(true)

	ScrollIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGray))
)
