package ui

import "github.com/charmbracelet/lipgloss"

// Brand colors
var (
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSuccess   = lipgloss.Color("#04B575") // Green
	ColorWarning   = lipgloss.Color("#FFBD2E") // Amber
	ColorError     = lipgloss.Color("#FF4672") // Red
	ColorInfo      = lipgloss.Color("#00BFFF") // Cyan
	ColorSubtle    = lipgloss.Color("#626262") // Gray
	ColorMuted     = lipgloss.Color("#4A4A4A") // Dark gray
	ColorHighlight = lipgloss.Color("#F1F1F1") // Near white
)

// Text styles
var (
	Bold      = lipgloss.NewStyle().Bold(true)
	Italic    = lipgloss.NewStyle().Italic(true)
	Faint     = lipgloss.NewStyle().Foreground(ColorSubtle)
	Highlight = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
)

// Status styles
var (
	SuccessStyle = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	InfoStyle    = lipgloss.NewStyle().Foreground(ColorInfo)
)

// Component styles
var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubheaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo)

	// TargetHeader is used for "--- [1/3] Processing mysql on 127.0.0.1:3306 ---"
	TargetHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				PaddingLeft(0)

	DriverBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorPrimary).
				Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)
)

// StatusIcon returns a styled status icon
func StatusIcon(success bool) string {
	if success {
		return SuccessStyle.Render("✓")
	}
	return ErrorStyle.Render("✗")
}

// DriverBadge returns a styled badge for a database driver name
func DriverBadge(driver string) string {
	return DriverBadgeStyle.Render(driver)
}
