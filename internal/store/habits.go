package store

import "github.com/MaximusBenjamin/terminal-pomodoro/internal/common"

func (s *Store) ListHabits() ([]common.Habit, error) {
	rows, err := s.db.Query("SELECT id, name, color, archived FROM habits WHERE archived = 0 ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []common.Habit
	for rows.Next() {
		var h common.Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Color, &h.Archived); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, rows.Err()
}

func (s *Store) AddHabit(name, color string) (int, error) {
	res, err := s.db.Exec("INSERT INTO habits (name, color) VALUES (?, ?)", name, color)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) DeleteHabit(id int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec("DELETE FROM sessions WHERE habit_id = ?", id)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM habits WHERE id = ?", id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetHabit(id int) (common.Habit, error) {
	var h common.Habit
	err := s.db.QueryRow("SELECT id, name, color, archived FROM habits WHERE id = ?", id).
		Scan(&h.ID, &h.Name, &h.Color, &h.Archived)
	return h, err
}
