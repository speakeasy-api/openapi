package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
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
	colorFooterText  = "#000000"
	colorWhite       = "#FFFFFF"
)

// methodColors maps HTTP methods to their display colors
var methodColors = map[string]lipgloss.Color{
	"GET":     colorGreen,
	"POST":    colorBlue,
	"PUT":     colorYellow,
	"DELETE":  colorRed,
	"PATCH":   colorPurple,
	"HEAD":    colorGray,
	"OPTIONS": colorGray,
	"TRACE":   colorGray,
}

// GetMethodColor returns the color for a given HTTP method
func GetMethodColor(method string) lipgloss.Color {
	if color, ok := methodColors[method]; ok {
		return color
	}
	return colorGray
}

// Common styles
var (
	// ButtonStyle is the default style for navigation buttons
	ButtonStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color(colorGray))

	// ActiveButtonStyle is the style for the active navigation button
	ActiveButtonStyle = ButtonStyle.
				Background(lipgloss.Color(colorThemePurple)).
				Foreground(lipgloss.Color(colorWhite)).
				Bold(true)

	// TitleStyle is the style for the app title
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorThemePurple))

	// DetailStyle is the style for detailed information
	DetailStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(colorDetailGray))

	// ScrollIndicatorStyle is the style for scroll indicators
	ScrollIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGray))

	// FooterStyle is the style for the footer
	FooterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(colorGray)).
			Foreground(lipgloss.Color(colorFooterText)).
			Padding(0, 1)

	// HelpKeyStyle is the style for keyboard shortcuts in help
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	// HelpTextStyle is the style for help text descriptions
	HelpTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite))

	// HelpModalStyle is the style for the help modal container
	HelpModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorThemePurple)).
			Padding(1, 2).
			Width(45)

	// HelpTitleStyle is the style for the help modal title
	HelpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorThemePurple)).
			Align(lipgloss.Center).
			Width(28)
)

// GetMethodStyle returns a styled method label
func GetMethodStyle(method string, highlighted bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Foreground(GetMethodColor(method)).
		Bold(true).
		Width(7)

	if highlighted {
		style = style.Background(lipgloss.Color(colorBackground))
	}

	return style
}

// GetHighlightStyle returns a style for highlighted items
func GetHighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(colorBackground))
}
