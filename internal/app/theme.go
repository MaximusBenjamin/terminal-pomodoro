package app

import (
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/theme"
)

// Re-export theme colors so existing code referencing app.ColorX continues to work.
var (
	ColorBg       = theme.ColorBg
	ColorFg       = theme.ColorFg
	ColorAccent   = theme.ColorAccent
	ColorOvertime = theme.ColorOvertime
	ColorSuccess  = theme.ColorSuccess
	ColorWarning  = theme.ColorWarning
	ColorBorder   = theme.ColorBorder
	ColorMuted    = theme.ColorMuted
	ColorBgLight  = theme.ColorBgLight
	ColorMagenta  = theme.ColorMagenta
	ColorCyan     = theme.ColorCyan
	ColorOrange   = theme.ColorOrange

	HeatmapColors = theme.HeatmapColors
)
