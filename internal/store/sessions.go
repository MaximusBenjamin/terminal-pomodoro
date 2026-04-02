package store

import "time"

type Session struct {
	ID              int
	HabitID         int
	StartTime       time.Time
	EndTime         time.Time
	PlannedMinutes  int
	ActualSeconds   int
	OvertimeSeconds int
	Completed       bool
}

func (s *Store) CreateSession(habitID, plannedMinutes, actualSeconds, overtimeSeconds int, completed bool) error {
	now := time.Now().Local()
	start := now.Add(-time.Duration(actualSeconds) * time.Second)

	// Store as clean format that SQLite's date() can parse
	const sqlFmt = "2006-01-02 15:04:05"

	_, err := s.db.Exec(
		`INSERT INTO sessions (habit_id, start_time, end_time, planned_minutes, actual_seconds, overtime_seconds, completed)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		habitID, start.Format(sqlFmt), now.Format(sqlFmt), plannedMinutes, actualSeconds, overtimeSeconds, boolToInt(completed),
	)
	return err
}

func (s *Store) RecentSessions(limit int) ([]Session, error) {
	rows, err := s.db.Query(
		`SELECT id, habit_id, start_time, end_time, planned_minutes, actual_seconds, overtime_seconds, completed
		 FROM sessions ORDER BY start_time DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var completed int
		if err := rows.Scan(&sess.ID, &sess.HabitID, &sess.StartTime, &sess.EndTime,
			&sess.PlannedMinutes, &sess.ActualSeconds, &sess.OvertimeSeconds, &completed); err != nil {
			return nil, err
		}
		sess.Completed = completed == 1
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

const sqlFmt = "2006-01-02 15:04:05"

// SessionWithHabit is a session joined with its habit name and color for display.
type SessionWithHabit struct {
	ID         int
	HabitID    int
	HabitName  string
	HabitColor string
	StartTime  time.Time
	EndTime    time.Time
	ActualSecs int
	Completed  bool
}

func (s *Store) ListSessionsWithHabits(limit int) ([]SessionWithHabit, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.habit_id, h.name, h.color, s.start_time, s.end_time, s.actual_seconds, s.completed
		 FROM sessions s
		 JOIN habits h ON h.id = s.habit_id
		 ORDER BY s.start_time DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SessionWithHabit
	for rows.Next() {
		var r SessionWithHabit
		var completed int
		if err := rows.Scan(&r.ID, &r.HabitID, &r.HabitName, &r.HabitColor,
			&r.StartTime, &r.EndTime, &r.ActualSecs, &completed); err != nil {
			return nil, err
		}
		r.Completed = completed == 1
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) DeleteSession(id int) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (s *Store) CreateManualSession(habitID int, startTime, endTime time.Time, actualSeconds int) error {
	plannedMinutes := actualSeconds / 60
	_, err := s.db.Exec(
		`INSERT INTO sessions (habit_id, start_time, end_time, planned_minutes, actual_seconds, overtime_seconds, completed)
		 VALUES (?, ?, ?, ?, ?, 0, 1)`,
		habitID, startTime.Format(sqlFmt), endTime.Format(sqlFmt), plannedMinutes, actualSeconds,
	)
	return err
}

func (s *Store) UpdateSession(id, habitID int, startTime, endTime time.Time, actualSeconds int) error {
	plannedMinutes := actualSeconds / 60
	_, err := s.db.Exec(
		`UPDATE sessions SET habit_id=?, start_time=?, end_time=?, planned_minutes=?, actual_seconds=?, overtime_seconds=0, completed=1
		 WHERE id=?`,
		habitID, startTime.Format(sqlFmt), endTime.Format(sqlFmt), plannedMinutes, actualSeconds, id,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
