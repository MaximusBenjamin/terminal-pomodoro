package theme

import "github.com/charmbracelet/lipgloss"

// Tokyo Night inspired color palette
var (
	ColorBg       = lipgloss.Color("#1a1b26")
	ColorFg       = lipgloss.Color("#a9b1d6")
	ColorAccent   = lipgloss.Color("#7aa2f7")
	ColorOvertime = lipgloss.Color("#f7768e")
	ColorSuccess  = lipgloss.Color("#9ece6a")
	ColorWarning  = lipgloss.Color("#e0af68")
	ColorBorder   = lipgloss.Color("#3b4261")
	ColorMuted    = lipgloss.Color("#565f89")
	ColorBgLight  = lipgloss.Color("#24283b")
	ColorMagenta  = lipgloss.Color("#bb9af7")
	ColorCyan     = lipgloss.Color("#7dcfff")
	ColorOrange   = lipgloss.Color("#ff9e64")

	// Heatmap intensity levels (dark to bright)
	HeatmapColors = []lipgloss.Color{
		lipgloss.Color("#1a1b26"), // 0 - none
		lipgloss.Color("#2f334d"), // 1 - low
		lipgloss.Color("#3d5999"), // 2 - medium-low
		lipgloss.Color("#5a7fcc"), // 3 - medium
		lipgloss.Color("#7aa2f7"), // 4 - high
	}
)
