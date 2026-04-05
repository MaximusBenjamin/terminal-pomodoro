package stats

import (
	"fmt"
	"time"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

// StreakResult holds the current streak and leeway state.
type StreakResult struct {
	CurrentStreak   int // days the streak has been active (complete + leeway days)
	LeewayUsedWeek  int // missed days consumed as leeway in the current ISO week
	LeewayRemaining int // leeway slots left this week
	LeewayPerWeek   int
}

// CalculateStreak walks backwards from yesterday computing the current streak.
// A day "counts" if all active habits have at least one session.
// Missed days consume leeway for their ISO week; exceeding leeway breaks the streak.
func CalculateStreak(
	dailySessions map[string]map[int]bool,
	habits []common.Habit,
	leewayPerWeek int,
) StreakResult {
	if len(habits) == 0 {
		return StreakResult{LeewayPerWeek: leewayPerWeek}
	}

	habitIDs := make(map[int]bool, len(habits))
	for _, h := range habits {
		habitIDs[h.ID] = true
	}

	now := time.Now().Local()
	today := api.EffectiveDate(now)

	// leewayUsed tracks missed days consumed per ISO week key "YYYY-WW"
	leewayUsed := make(map[string]int)

	streak := 0
	leewayUsedThisWeek := 0

	// Walk backwards from yesterday — today is still in progress
	for offset := 1; offset <= 365; offset++ {
		day := today.AddDate(0, 0, -offset)
		key := day.Format("2006-01-02")
		weekKey := day.Format("2006-W") + weekNum(day)

		sessions := dailySessions[key]
		dayComplete := isDayComplete(sessions, habitIDs)

		if dayComplete {
			streak++
		} else {
			// Day was missed — check leeway for this week
			if leewayUsed[weekKey] < leewayPerWeek {
				leewayUsed[weekKey]++
				streak++ // leeway day still counts toward streak length
			} else {
				// Leeway exhausted — streak is broken
				break
			}
		}

		// Track leeway used in the current ISO week (offset=1 is yesterday)
		yearCurrent, weekCurrent := today.ISOWeek()
		yearDay, weekDay := day.ISOWeek()
		if yearDay == yearCurrent && weekDay == weekCurrent {
			leewayUsedThisWeek = leewayUsed[weekKey]
		}
	}

	leewayRemaining := leewayPerWeek - leewayUsedThisWeek
	if leewayRemaining < 0 {
		leewayRemaining = 0
	}

	return StreakResult{
		CurrentStreak:   streak,
		LeewayUsedWeek:  leewayUsedThisWeek,
		LeewayRemaining: leewayRemaining,
		LeewayPerWeek:   leewayPerWeek,
	}
}

// isDayComplete returns true if all required habit IDs have sessions on that day.
func isDayComplete(sessions map[int]bool, habitIDs map[int]bool) bool {
	if len(sessions) == 0 {
		return false
	}
	for id := range habitIDs {
		if !sessions[id] {
			return false
		}
	}
	return true
}

func weekNum(t time.Time) string {
	_, week := t.ISOWeek()
	return fmt.Sprintf("%02d", week)
}
