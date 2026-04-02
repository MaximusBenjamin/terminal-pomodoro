package stats

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// RenderSummary renders a horizontal bar: Today: X.Xh │ This Week: X.Xh │ All-Time: X.Xh
func RenderSummary(today, week, allTime float64, width int) string {
	sep := common.MutedStyle.Render("  │  ")

	todayStr := common.MutedStyle.Render("Today: ") + common.AccentStyle.Render(fmtDuration(today))
	weekStr := common.MutedStyle.Render("This Week: ") + common.AccentStyle.Render(fmtDuration(week))
	allTimeStr := common.MutedStyle.Render("All-Time: ") + common.AccentStyle.Render(fmtDuration(allTime))

	line := todayStr + sep + weekStr + sep + allTimeStr

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(line)
}

// fmtDuration formats hours intelligently: shows minutes for small values.
func fmtDuration(hours float64) string {
	if hours == 0 {
		return "0m"
	}
	if hours < 0.1 {
		m := int(math.Round(hours * 60))
		if m == 0 {
			s := int(math.Round(hours * 3600))
			return fmt.Sprintf("%ds", s)
		}
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%.1fh", hours)
}
