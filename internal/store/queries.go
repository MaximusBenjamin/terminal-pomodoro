package store

import (
	"strconv"
	"time"
)

type DailyHours struct {
	Date  time.Time
	Hours float64
}

type HabitBreakdown struct {
	HabitID   int
	HabitName string
	Color     string
	Hours     float64
}

// TodayHours returns total hours studied today.
func (s *Store) TodayHours() (float64, error) {
	var seconds float64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(actual_seconds), 0) FROM sessions
		 WHERE date(start_time) = date('now', 'localtime')`,
	).Scan(&seconds)
	return seconds / 3600.0, err
}

// WeekHours returns total hours studied this week (Mon-Sun).
func (s *Store) WeekHours() (float64, error) {
	var seconds float64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(actual_seconds), 0) FROM sessions
		 WHERE start_time >= date('now', 'localtime', 'weekday 1', '-7 days')`,
	).Scan(&seconds)
	return seconds / 3600, err
}

// AllTimeHours returns total hours studied ever.
func (s *Store) AllTimeHours() (float64, error) {
	var seconds float64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(actual_seconds), 0) FROM sessions`,
	).Scan(&seconds)
	return seconds / 3600, err
}

// DailyHoursRange returns hours per day for the last N days.
func (s *Store) DailyHoursRange(days int) ([]DailyHours, error) {
	rows, err := s.db.Query(
		`SELECT date(start_time, 'localtime') as d, SUM(actual_seconds) / 3600.0
		 FROM sessions
		 WHERE start_time >= date('now', 'localtime', ?)
		 GROUP BY d
		 ORDER BY d`,
		"-"+strconv.Itoa(days)+" days",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build a map of date -> hours
	hoursMap := make(map[string]float64)
	for rows.Next() {
		var dateStr string
		var hours float64
		if err := rows.Scan(&dateStr, &hours); err != nil {
			return nil, err
		}
		hoursMap[dateStr] = hours
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fill in all days
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	result := make([]DailyHours, days)
	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -(days-1-i))
		dateStr := d.Format("2006-01-02")
		result[i] = DailyHours{
			Date:  d,
			Hours: hoursMap[dateStr],
		}
	}
	return result, nil
}

// mondayOfWeek returns the Monday of the current ISO week.
func mondayOfWeek() time.Time {
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	wd := today.Weekday()
	daysSinceMonday := int(wd) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}
	return today.AddDate(0, 0, -daysSinceMonday)
}

// WeekDailyHours returns hours for Mon-Sun of the current week.
// Index 0=Mon, 1=Tue, ..., 6=Sun.
func (s *Store) WeekDailyHours() ([]float64, error) {
	monday := mondayOfWeek()
	sunday := monday.AddDate(0, 0, 6)

	rows, err := s.db.Query(
		`SELECT date(start_time, 'localtime') as d, SUM(actual_seconds) / 3600.0
		 FROM sessions
		 WHERE date(start_time, 'localtime') >= ? AND date(start_time, 'localtime') <= ?
		 GROUP BY d ORDER BY d`,
		monday.Format("2006-01-02"), sunday.Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hoursMap := make(map[string]float64)
	for rows.Next() {
		var dateStr string
		var hours float64
		if err := rows.Scan(&dateStr, &hours); err != nil {
			return nil, err
		}
		hoursMap[dateStr] = hours
	}

	result := make([]float64, 7)
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		result[i] = hoursMap[d.Format("2006-01-02")]
	}
	return result, rows.Err()
}

// TodayHoursByHabit returns hours per habit for today.
func (s *Store) TodayHoursByHabit() ([]HabitBreakdown, error) {
	rows, err := s.db.Query(
		`SELECT h.id, h.name, h.color, COALESCE(SUM(s.actual_seconds), 0) / 3600.0
		 FROM habits h
		 LEFT JOIN sessions s ON s.habit_id = h.id
		   AND date(s.start_time, 'localtime') = date('now', 'localtime')
		 WHERE h.archived = 0
		 GROUP BY h.id
		 ORDER BY h.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []HabitBreakdown
	for rows.Next() {
		var hb HabitBreakdown
		if err := rows.Scan(&hb.HabitID, &hb.HabitName, &hb.Color, &hb.Hours); err != nil {
			return nil, err
		}
		result = append(result, hb)
	}
	return result, rows.Err()
}

// WeekDailyByHabit returns per-habit hours for Mon-Sun of the current week.
// Returns a map of habit_id -> HabitWeekData with Daily[0]=Mon, Daily[6]=Sun.
func (s *Store) WeekDailyByHabit() (map[int]HabitWeekData, error) {
	monday := mondayOfWeek()
	sunday := monday.AddDate(0, 0, 6)

	rows, err := s.db.Query(
		`SELECT h.id, h.name, h.color, date(s.start_time, 'localtime') as d,
		        COALESCE(SUM(s.actual_seconds), 0) / 3600.0
		 FROM habits h
		 LEFT JOIN sessions s ON s.habit_id = h.id
		   AND date(s.start_time, 'localtime') >= ?
		   AND date(s.start_time, 'localtime') <= ?
		 WHERE h.archived = 0
		 GROUP BY h.id, d
		 ORDER BY h.name, d`,
		monday.Format("2006-01-02"), sunday.Format("2006-01-02"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]HabitWeekData)
	for rows.Next() {
		var id int
		var name, color string
		var dateStr *string
		var hours float64
		if err := rows.Scan(&id, &name, &color, &dateStr, &hours); err != nil {
			return nil, err
		}
		hw := result[id]
		hw.HabitID = id
		hw.HabitName = name
		hw.Color = color
		if dateStr != nil {
			d, err := time.Parse("2006-01-02", *dateStr)
			if err == nil {
				dayIdx := int(d.Sub(monday).Hours() / 24)
				if dayIdx >= 0 && dayIdx < 7 {
					hw.Daily[dayIdx] = hours
				}
			}
		}
		result[id] = hw
	}
	return result, rows.Err()
}

// HabitWeekData holds per-day hours for a single habit over 7 days.
type HabitWeekData struct {
	HabitID   int
	HabitName string
	Color     string
	Daily     [7]float64 // index 0=Mon, 1=Tue, ..., 6=Sun
}

// HabitBreakdownForPeriod returns hours per habit for the last N days.
func (s *Store) HabitBreakdownForPeriod(days int) ([]HabitBreakdown, error) {
	rows, err := s.db.Query(
		`SELECT h.id, h.name, h.color, COALESCE(SUM(s.actual_seconds), 0) / 3600.0
		 FROM habits h
		 LEFT JOIN sessions s ON s.habit_id = h.id
		   AND s.start_time >= date('now', 'localtime', ?)
		 WHERE h.archived = 0
		 GROUP BY h.id
		 HAVING SUM(s.actual_seconds) > 0
		 ORDER BY SUM(s.actual_seconds) DESC`,
		"-"+strconv.Itoa(days)+" days",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []HabitBreakdown
	for rows.Next() {
		var hb HabitBreakdown
		if err := rows.Scan(&hb.HabitID, &hb.HabitName, &hb.Color, &hb.Hours); err != nil {
			return nil, err
		}
		result = append(result, hb)
	}
	return result, rows.Err()
}

