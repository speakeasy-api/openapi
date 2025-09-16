package navigator

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the styling for the navigator
type Styles struct {
	// Header styles
	HeaderStyle     lipgloss.Style
	BreadcrumbStyle lipgloss.Style

	// Content styles
	ContentStyle     lipgloss.Style
	ItemStyle        lipgloss.Style
	SelectedStyle    lipgloss.Style
	PreviewStyle     lipgloss.Style
	DescriptionStyle lipgloss.Style

	// Footer styles
	FooterStyle lipgloss.Style
	HelpStyle   lipgloss.Style

	// Colors
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	AccentColor    lipgloss.Color
	MutedColor     lipgloss.Color
	ErrorColor     lipgloss.Color
	SuccessColor   lipgloss.Color
}

// NewStyles creates a new set of styles with default values
func NewStyles() *Styles {
	s := &Styles{
		PrimaryColor:   lipgloss.Color("#7C3AED"), // Purple
		SecondaryColor: lipgloss.Color("#6B7280"), // Gray
		AccentColor:    lipgloss.Color("#10B981"), // Green
		MutedColor:     lipgloss.Color("#9CA3AF"), // Light gray
		ErrorColor:     lipgloss.Color("#EF4444"), // Red
		SuccessColor:   lipgloss.Color("#10B981"), // Green
	}

	// Header styles - removed background
	s.HeaderStyle = lipgloss.NewStyle().
		Foreground(s.PrimaryColor).
		Padding(0, 1).
		Bold(true)

	s.BreadcrumbStyle = lipgloss.NewStyle().
		Foreground(s.PrimaryColor).
		Bold(true)

	// Content styles
	s.ContentStyle = lipgloss.NewStyle().
		Padding(1, 2)

	s.ItemStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0, 0, 0, 0)

	s.SelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#374151")). // Dark grey background
		Foreground(lipgloss.Color("#FFFFFF")). // White text for contrast
		Bold(true)

	s.PreviewStyle = lipgloss.NewStyle().
		Foreground(s.MutedColor).
		Padding(0, 0, 0, 4)

	s.DescriptionStyle = lipgloss.NewStyle().
		Foreground(s.SecondaryColor).
		Italic(true)

	// Footer styles
	s.FooterStyle = lipgloss.NewStyle().
		Background(s.SecondaryColor).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	s.HelpStyle = lipgloss.NewStyle().
		Foreground(s.MutedColor)

	return s
}

// HTTPMethodColor returns the appropriate color for HTTP methods
func (s *Styles) HTTPMethodColor(method string) lipgloss.Color {
	switch method {
	case "GET":
		return lipgloss.Color("#10B981") // Green
	case "POST":
		return lipgloss.Color("#3B82F6") // Blue
	case "PUT":
		return lipgloss.Color("#F59E0B") // Orange
	case "PATCH":
		return lipgloss.Color("#8B5CF6") // Purple
	case "DELETE":
		return lipgloss.Color("#EF4444") // Red
	case "HEAD":
		return lipgloss.Color("#6B7280") // Gray
	case "OPTIONS":
		return lipgloss.Color("#6B7280") // Gray
	default:
		return s.SecondaryColor
	}
}

// HTTPMethodStyle returns a styled HTTP method
func (s *Styles) HTTPMethodStyle(method string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(s.HTTPMethodColor(method)).
		Bold(true).
		Width(7). // Fixed width for alignment
		Align(lipgloss.Left)
}

// StatusCodeColor returns the appropriate color for HTTP status codes
func (s *Styles) StatusCodeColor(code string) lipgloss.Color {
	if len(code) == 0 {
		return s.SecondaryColor
	}

	switch code[0] {
	case '2':
		return s.SuccessColor // 2xx - Success
	case '3':
		return lipgloss.Color("#3B82F6") // 3xx - Redirection (Blue)
	case '4':
		return lipgloss.Color("#F59E0B") // 4xx - Client Error (Orange)
	case '5':
		return s.ErrorColor // 5xx - Server Error (Red)
	default:
		return s.SecondaryColor
	}
}

// NodeTypeStyle returns a style for different node types
func (s *Styles) NodeTypeStyle(nodeType NodeType) lipgloss.Style {
	switch nodeType {
	case NodeTypeRoot:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true) // Red
	case NodeTypeInfo:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#2563EB")).Bold(true) // Blue
	case NodeTypeServers:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#059669")).Bold(true) // Green
	case NodeTypeServer:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#059669")) // Green
	case NodeTypePaths:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true) // Purple
	case NodeTypePath:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")) // Purple
	case NodeTypeOperation:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EA580C")).Bold(true) // Orange
	case NodeTypeComponents:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#0891B2")).Bold(true) // Cyan
	case NodeTypeSchemas:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Bold(true) // Purple
	case NodeTypeSchema:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")) // Purple
	case NodeTypeParameters:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true) // Amber
	case NodeTypeParameter:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")) // Amber
	case NodeTypeResponses:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true) // Emerald
	case NodeTypeResponse:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")) // Emerald
	case NodeTypeRequestBody:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EC4899")).Bold(true) // Pink
	case NodeTypeSecurity:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#DC2626")).Bold(true) // Red
	case NodeTypeTags:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6366F1")).Bold(true) // Indigo
	case NodeTypeTag:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6366F1")) // Indigo
	default:
		return lipgloss.NewStyle().Foreground(s.SecondaryColor)
	}
}
