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

	// Heatmap intensity levels (dark green to bright green)
	HeatmapColors = []lipgloss.Color{
		lipgloss.Color("#3b4261"), // 0 - none (muted gray)
		lipgloss.Color("#1e3a1e"), // 1 - low
		lipgloss.Color("#2d6b2d"), // 2 - medium-low
		lipgloss.Color("#52a852"), // 3 - medium
		lipgloss.Color("#9ece6a"), // 4 - high (bright green)
	}
)
