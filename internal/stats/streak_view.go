package stats

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/theme"
)

// RenderStreak renders the streak counter and weekly leeway indicator.
func RenderStreak(s StreakResult, width int) string {
	if s.LeewayPerWeek == 0 && s.CurrentStreak == 0 {
		// Streak with no leeway, nothing yet
		return renderStreakLine(s)
	}
	return renderStreakLine(s) + "\n" + renderLeewayBar(s, width)
}

func renderStreakLine(s StreakResult) string {
	streakStyle := lipgloss.NewStyle().Foreground(theme.ColorSuccess).Bold(true)
	labelStyle := common.MutedStyle

	var streakStr string
	if s.CurrentStreak == 0 {
		streakStr = labelStyle.Render("Streak: ") + common.MutedStyle.Render("—  start today!")
	} else {
		flame := streakStyle.Render(fmt.Sprintf("%d", s.CurrentStreak))
		unit := labelStyle.Render(" day")
		if s.CurrentStreak != 1 {
			unit = labelStyle.Render(" days")
		}
		streakStr = labelStyle.Render("Streak: ") + flame + unit
	}
	return streakStr
}

func renderLeewayBar(s StreakResult, width int) string {
	if s.LeewayPerWeek == 0 {
		return ""
	}

	usedStyle := lipgloss.NewStyle().Foreground(theme.ColorWarning)
	freeStyle := lipgloss.NewStyle().Foreground(theme.ColorSuccess)
	labelStyle := common.MutedStyle

	// Build pip indicators: ● used leeway, ○ remaining
	var pips []string
	for i := 0; i < s.LeewayPerWeek; i++ {
		if i < s.LeewayUsedWeek {
			pips = append(pips, usedStyle.Render("●"))
		} else {
			pips = append(pips, freeStyle.Render("○"))
		}
	}
	pipStr := strings.Join(pips, " ")

	label := labelStyle.Render(fmt.Sprintf("Leeway this week: %s  (%d/%d used)",
		pipStr, s.LeewayUsedWeek, s.LeewayPerWeek))
	return label
}
