package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

const (
	supabaseURL     = "https://dqnvsgtksqhbrmqchlds.supabase.co"
	supabaseAnonKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImRxbnZzZ3Rrc3FoYnJtcWNobGRzIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzUxNjQxNDMsImV4cCI6MjA5MDc0MDE0M30.ew7OMJlZQDdpi1d7Mfe6s7kcA6wuNtAIZxFGNdVBxjw"
)

// Client is a Supabase REST API client that provides the same interface as store.Store.
type Client struct {
	baseURL      string
	anonKey      string
	authToken    string
	refreshToken string
	httpClient   *http.Client
}

// NewClient creates a new Supabase API client with the stored auth token.
// It automatically refreshes an expired token using the refresh token.
func NewClient() (*Client, error) {
	auth, err := LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w", err)
	}

	// Try refreshing the token proactively to avoid 401s
	if auth.RefreshToken != "" {
		if refreshed, err := RefreshAccessToken(); err == nil {
			_ = SaveAuth(refreshed)
			auth = refreshed
		}
	}

	return &Client{
		baseURL:      supabaseURL,
		anonKey:      supabaseAnonKey,
		authToken:    auth.AccessToken,
		refreshToken: auth.RefreshToken,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// NewClientWithToken creates a client with a specific auth token (used during login/register).
func NewClientWithToken(token string) *Client {
	return &Client{
		baseURL:    supabaseURL,
		anonKey:    supabaseAnonKey,
		authToken:  token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Close is a no-op for the API client (satisfies the same pattern as store.Store).
func (c *Client) Close() error {
	return nil
}

// doRequest performs an HTTP request to the Supabase REST API.
func (c *Client) doRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 401 && c.refreshToken != "" {
		// Token expired — try refreshing and retry once
		refreshed, err := refreshWithToken(c.refreshToken)
		if err != nil {
			return nil, fmt.Errorf("session expired, please run: tpom login")
		}
		_ = SaveAuth(refreshed)
		c.authToken = refreshed.AccessToken
		c.refreshToken = refreshed.RefreshToken

		// Rebuild the request with new token
		var retryReader io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			retryReader = bytes.NewReader(b)
		}
		retryReq, err := http.NewRequest(method, c.baseURL+path, retryReader)
		if err != nil {
			return nil, fmt.Errorf("creating retry request: %w", err)
		}
		retryReq.Header.Set("apikey", c.anonKey)
		retryReq.Header.Set("Authorization", "Bearer "+c.authToken)
		retryReq.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			retryReq.Header.Set(k, v)
		}

		retryResp, err := c.httpClient.Do(retryReq)
		if err != nil {
			return nil, fmt.Errorf("executing retry request: %w", err)
		}
		defer retryResp.Body.Close()

		data, err = io.ReadAll(retryResp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading retry response: %w", err)
		}
		if retryResp.StatusCode < 200 || retryResp.StatusCode >= 300 {
			return nil, fmt.Errorf("API error %d: %s", retryResp.StatusCode, string(data))
		}
		return data, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// --- Habit types and methods ---

type apiHabit struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Archived bool   `json:"archived"`
}

func (c *Client) ListHabits() ([]common.Habit, error) {
	data, err := c.doRequest("GET", "/rest/v1/habits?archived=eq.false&order=name", nil, nil)
	if err != nil {
		return nil, err
	}

	var apiHabits []apiHabit
	if err := json.Unmarshal(data, &apiHabits); err != nil {
		return nil, fmt.Errorf("decoding habits: %w", err)
	}

	habits := make([]common.Habit, len(apiHabits))
	for i, h := range apiHabits {
		habits[i] = common.Habit{
			ID:       h.ID,
			Name:     h.Name,
			Color:    h.Color,
			Archived: h.Archived,
		}
	}
	return habits, nil
}

func (c *Client) AddHabit(name, color string) (int, error) {
	body := map[string]interface{}{
		"name":  name,
		"color": color,
	}
	headers := map[string]string{
		"Prefer": "return=representation",
	}
	data, err := c.doRequest("POST", "/rest/v1/habits", body, headers)
	if err != nil {
		return 0, err
	}

	var result []apiHabit
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("decoding created habit: %w", err)
	}
	if len(result) == 0 {
		return 0, fmt.Errorf("no habit returned after insert")
	}
	return result[0].ID, nil
}

func (c *Client) DeleteHabit(id int) error {
	// Delete sessions first, then the habit (PostgREST handles cascade, but be explicit)
	path := fmt.Sprintf("/rest/v1/habits?id=eq.%d", id)
	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

func (c *Client) GetHabit(id int) (common.Habit, error) {
	path := fmt.Sprintf("/rest/v1/habits?id=eq.%d", id)
	data, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return common.Habit{}, err
	}

	var apiHabits []apiHabit
	if err := json.Unmarshal(data, &apiHabits); err != nil {
		return common.Habit{}, fmt.Errorf("decoding habit: %w", err)
	}
	if len(apiHabits) == 0 {
		return common.Habit{}, fmt.Errorf("habit not found")
	}

	h := apiHabits[0]
	return common.Habit{
		ID:       h.ID,
		Name:     h.Name,
		Color:    h.Color,
		Archived: h.Archived,
	}, nil
}

// --- Session types and methods ---

type apiSession struct {
	ID              int        `json:"id"`
	HabitID         int        `json:"habit_id"`
	PlannedMinutes  int        `json:"planned_minutes"`
	ActualSeconds   int        `json:"actual_seconds"`
	OvertimeSeconds int        `json:"overtime_seconds"`
	Completed       bool       `json:"completed"`
	StartTime       time.Time  `json:"start_time"`
	CreatedAt       time.Time  `json:"created_at"`
	Habit           *apiHabit  `json:"habits,omitempty"` // joined via PostgREST select
}

// SessionWithHabit mirrors store.SessionWithHabit for display in the log view.
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

func (c *Client) CreateSession(habitID, plannedMinutes, actualSeconds, overtimeSeconds int, completed bool) error {
	now := time.Now().Local()
	start := now.Add(-time.Duration(actualSeconds) * time.Second)

	body := map[string]interface{}{
		"habit_id":         habitID,
		"planned_minutes":  plannedMinutes,
		"actual_seconds":   actualSeconds,
		"overtime_seconds": overtimeSeconds,
		"completed":        completed,
		"start_time":       start.Format(time.RFC3339),
	}
	_, err := c.doRequest("POST", "/rest/v1/sessions", body, nil)
	return err
}

func (c *Client) CreateManualSession(habitID int, startTime, endTime time.Time, actualSeconds int) error {
	plannedMinutes := actualSeconds / 60
	body := map[string]interface{}{
		"habit_id":         habitID,
		"planned_minutes":  plannedMinutes,
		"actual_seconds":   actualSeconds,
		"overtime_seconds": 0,
		"completed":        true,
		"start_time":       startTime.Format(time.RFC3339),
	}
	_, err := c.doRequest("POST", "/rest/v1/sessions", body, nil)
	return err
}

func (c *Client) UpdateSession(id, habitID int, startTime, endTime time.Time, actualSeconds int) error {
	plannedMinutes := actualSeconds / 60
	body := map[string]interface{}{
		"habit_id":         habitID,
		"planned_minutes":  plannedMinutes,
		"actual_seconds":   actualSeconds,
		"overtime_seconds": 0,
		"completed":        true,
		"start_time":       startTime.Format(time.RFC3339),
	}
	path := fmt.Sprintf("/rest/v1/sessions?id=eq.%d", id)
	_, err := c.doRequest("PATCH", path, body, nil)
	return err
}

func (c *Client) DeleteSession(id int) error {
	path := fmt.Sprintf("/rest/v1/sessions?id=eq.%d", id)
	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

func (c *Client) ListSessionsWithHabits(limit int) ([]SessionWithHabit, error) {
	path := "/rest/v1/sessions?select=*,habits(name,color)&order=start_time.desc&limit=" + url.QueryEscape(strconv.Itoa(limit))
	data, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var apiSessions []apiSession
	if err := json.Unmarshal(data, &apiSessions); err != nil {
		return nil, fmt.Errorf("decoding sessions: %w", err)
	}

	result := make([]SessionWithHabit, len(apiSessions))
	for i, s := range apiSessions {
		habitName := ""
		habitColor := ""
		if s.Habit != nil {
			habitName = s.Habit.Name
			habitColor = s.Habit.Color
		}
		endTime := s.StartTime.Add(time.Duration(s.ActualSeconds) * time.Second)
		result[i] = SessionWithHabit{
			ID:         s.ID,
			HabitID:    s.HabitID,
			HabitName:  habitName,
			HabitColor: habitColor,
			StartTime:  s.StartTime.Local(),
			EndTime:    endTime.Local(),
			ActualSecs: s.ActualSeconds,
			Completed:  s.Completed,
		}
	}
	return result, nil
}

// --- Aggregate query types (mirror store types) ---

// DailyHours represents hours for a single day.
type DailyHours struct {
	Date  time.Time
	Hours float64
}

// HabitBreakdown represents hours for a single habit in a period.
type HabitBreakdown struct {
	HabitID   int
	HabitName string
	Color     string
	Hours     float64
}

// HabitWeekData holds per-day hours for a single habit over 7 days.
type HabitWeekData struct {
	HabitID   int
	HabitName string
	Color     string
	Daily     [7]float64 // index 0=Mon, 1=Tue, ..., 6=Sun
}

// EffectiveDate returns the "logical" date for a given time, where the day
// boundary is at 4 AM instead of midnight. A session at 2 AM on April 4 is
// considered to belong to April 3.
func EffectiveDate(t time.Time) time.Time {
	shifted := t.Add(-4 * time.Hour)
	return time.Date(shifted.Year(), shifted.Month(), shifted.Day(), 0, 0, 0, 0, shifted.Location())
}

// MondayOfWeek returns the Monday of the current ISO week offset by n weeks.
// 0 = current week, -1 = last week, etc. Uses the 4 AM day boundary.
func MondayOfWeek(offset int) time.Time {
	now := time.Now().Local()
	today := EffectiveDate(now)
	wd := today.Weekday()
	daysSinceMonday := int(wd) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}
	return today.AddDate(0, 0, -daysSinceMonday+(offset*7))
}

// fetchAllSessions retrieves all sessions from Supabase for aggregate queries.
func (c *Client) fetchAllSessions() ([]apiSession, error) {
	data, err := c.doRequest("GET", "/rest/v1/sessions?select=*,habits(name,color)&order=start_time.desc", nil, nil)
	if err != nil {
		return nil, err
	}

	var sessions []apiSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("decoding sessions: %w", err)
	}
	return sessions, nil
}

// TodayHours returns total hours studied today (day starts at 4 AM).
func (c *Client) TodayHours() (float64, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return 0, err
	}

	now := time.Now().Local()
	today := EffectiveDate(now)
	var totalSeconds float64

	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if sessionDate.Equal(today) {
			totalSeconds += float64(s.ActualSeconds)
		}
	}
	return totalSeconds / 3600.0, nil
}

// WeekHours returns total hours studied this week (Mon-Sun, day starts at 4 AM).
func (c *Client) WeekHours() (float64, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return 0, err
	}

	monday := MondayOfWeek(0)
	var totalSeconds float64

	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if !sessionDate.Before(monday) {
			totalSeconds += float64(s.ActualSeconds)
		}
	}
	return totalSeconds / 3600.0, nil
}

// AllTimeHours returns total hours studied ever.
func (c *Client) AllTimeHours() (float64, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return 0, err
	}

	var totalSeconds float64
	for _, s := range sessions {
		totalSeconds += float64(s.ActualSeconds)
	}
	return totalSeconds / 3600.0, nil
}

// DailyHoursRange returns hours per day for the last N days (day starts at 4 AM).
func (c *Client) DailyHoursRange(days int) ([]DailyHours, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	now := time.Now().Local()
	today := EffectiveDate(now)
	startDate := today.AddDate(0, 0, -(days - 1))

	// Build a map of date -> total seconds
	hoursMap := make(map[string]float64)
	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if !sessionDate.Before(startDate) && !sessionDate.After(today) {
			dateStr := sessionDate.Format("2006-01-02")
			hoursMap[dateStr] += float64(s.ActualSeconds)
		}
	}

	// Fill in all days
	result := make([]DailyHours, days)
	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -(days-1-i))
		dateStr := d.Format("2006-01-02")
		result[i] = DailyHours{
			Date:  d,
			Hours: hoursMap[dateStr] / 3600.0,
		}
	}
	return result, nil
}

// WeekDailyHours returns hours for Mon-Sun of the given week offset.
// offset 0 = current week, -1 = last week, etc. Index 0=Mon, 6=Sun.
func (c *Client) WeekDailyHours(offset int) ([]float64, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	monday := MondayOfWeek(offset)
	sunday := monday.AddDate(0, 0, 6)

	hoursMap := make(map[string]float64)
	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if !sessionDate.Before(monday) && !sessionDate.After(sunday) {
			dateStr := sessionDate.Format("2006-01-02")
			hoursMap[dateStr] += float64(s.ActualSeconds)
		}
	}

	result := make([]float64, 7)
	for i := 0; i < 7; i++ {
		d := monday.AddDate(0, 0, i)
		result[i] = hoursMap[d.Format("2006-01-02")] / 3600.0
	}
	return result, nil
}

// TodayHoursByHabit returns hours per habit for today.
func (c *Client) TodayHoursByHabit() ([]HabitBreakdown, error) {
	habits, err := c.ListHabits()
	if err != nil {
		return nil, err
	}

	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	now := time.Now().Local()
	today := EffectiveDate(now)

	// Accumulate seconds per habit
	habitSeconds := make(map[int]float64)
	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if sessionDate.Equal(today) {
			habitSeconds[s.HabitID] += float64(s.ActualSeconds)
		}
	}

	// Build result for all non-archived habits
	var result []HabitBreakdown
	for _, h := range habits {
		result = append(result, HabitBreakdown{
			HabitID:   h.ID,
			HabitName: h.Name,
			Color:     h.Color,
			Hours:     habitSeconds[h.ID] / 3600.0,
		})
	}
	return result, nil
}

// WeekDailyByHabit returns per-habit hours for Mon-Sun of the given week offset.
func (c *Client) WeekDailyByHabit(offset int) (map[int]HabitWeekData, error) {
	habits, err := c.ListHabits()
	if err != nil {
		return nil, err
	}

	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	monday := MondayOfWeek(offset)
	sunday := monday.AddDate(0, 0, 6)

	result := make(map[int]HabitWeekData)

	// Initialize all habits in result
	for _, h := range habits {
		result[h.ID] = HabitWeekData{
			HabitID:   h.ID,
			HabitName: h.Name,
			Color:     h.Color,
		}
	}

	// Accumulate session seconds into days
	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if !sessionDate.Before(monday) && !sessionDate.After(sunday) {
			hw := result[s.HabitID]
			if hw.HabitName == "" {
				// Session for an archived/deleted habit; skip
				continue
			}
			dayIdx := int(sessionDate.Sub(monday).Hours() / 24)
			if dayIdx >= 0 && dayIdx < 7 {
				hw.Daily[dayIdx] += float64(s.ActualSeconds) / 3600.0
			}
			result[s.HabitID] = hw
		}
	}

	return result, nil
}

// DailyHabitSessions returns a map of date string → set of habit IDs that had
// at least one session on that day (using 4AM boundary), for the last N days.
func (c *Client) DailyHabitSessions(days int) (map[string]map[int]bool, error) {
	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[int]bool)
	now := time.Now().Local()
	today := EffectiveDate(now)
	startDate := today.AddDate(0, 0, -days)

	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if sessionDate.Before(startDate) || sessionDate.After(today) {
			continue
		}
		key := sessionDate.Format("2006-01-02")
		if result[key] == nil {
			result[key] = make(map[int]bool)
		}
		result[key][s.HabitID] = true
	}
	return result, nil
}

// HabitBreakdownForPeriod returns hours per habit for the last N days.
func (c *Client) HabitBreakdownForPeriod(days int) ([]HabitBreakdown, error) {
	habits, err := c.ListHabits()
	if err != nil {
		return nil, err
	}

	sessions, err := c.fetchAllSessions()
	if err != nil {
		return nil, err
	}

	now := time.Now().Local()
	today := EffectiveDate(now)
	startDate := today.AddDate(0, 0, -days)

	// Accumulate seconds per habit
	habitSeconds := make(map[int]float64)
	for _, s := range sessions {
		sessionDate := EffectiveDate(s.StartTime.Local())
		if sessionDate.After(startDate) && !sessionDate.After(today) {
			habitSeconds[s.HabitID] += float64(s.ActualSeconds)
		}
	}

	// Build result for habits that have data
	var result []HabitBreakdown
	for _, h := range habits {
		secs := habitSeconds[h.ID]
		if secs > 0 {
			result = append(result, HabitBreakdown{
				HabitID:   h.ID,
				HabitName: h.Name,
				Color:     h.Color,
				Hours:     secs / 3600.0,
			})
		}
	}

	// Sort by hours descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Hours > result[j].Hours
	})

	return result, nil
}

// GetLeeway fetches the leeway_days_per_week setting from Supabase.
func (c *Client) GetLeeway() (int, error) {
	data, err := c.doRequest("GET", "/rest/v1/user_settings?select=leeway_days_per_week", nil, map[string]string{
		"Accept": "application/vnd.pgrst.object+json",
	})
	if err != nil {
		return 0, err
	}
	var row struct {
		Leeway int `json:"leeway_days_per_week"`
	}
	if err := json.Unmarshal(data, &row); err != nil {
		return 0, err
	}
	return row.Leeway, nil
}

// SetLeeway upserts the leeway_days_per_week setting in Supabase.
func (c *Client) SetLeeway(leeway int) error {
	body := map[string]interface{}{
		"leeway_days_per_week": leeway,
		"updated_at":           time.Now().UTC().Format(time.RFC3339),
	}
	_, err := c.doRequest("POST", "/rest/v1/user_settings", body, map[string]string{
		"Prefer": "resolution=merge-duplicates",
	})
	return err
}

// --- Todo types and methods ---

type apiTodo struct {
	ID            int        `json:"id"`
	Text          string     `json:"text"`
	Completed     bool       `json:"completed"`
	EffectiveDate string     `json:"effective_date"` // YYYY-MM-DD
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

func (a apiTodo) toCommon() (common.Todo, error) {
	day, err := time.ParseInLocation("2006-01-02", a.EffectiveDate, time.Local)
	if err != nil {
		return common.Todo{}, fmt.Errorf("parsing effective_date %q: %w", a.EffectiveDate, err)
	}
	return common.Todo{
		ID:            a.ID,
		Text:          a.Text,
		Completed:     a.Completed,
		EffectiveDate: day,
		CreatedAt:     a.CreatedAt,
		CompletedAt:   a.CompletedAt,
	}, nil
}

// ListTodos returns the user's todos for the given effective date (a date-only
// time.Time, typically from EffectiveDate(time.Now())).
func (c *Client) ListTodos(date time.Time) ([]common.Todo, error) {
	dateStr := date.Format("2006-01-02")
	path := fmt.Sprintf("/rest/v1/todos?effective_date=eq.%s&order=created_at.asc", dateStr)
	data, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var rows []apiTodo
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("decoding todos: %w", err)
	}

	todos := make([]common.Todo, 0, len(rows))
	for _, r := range rows {
		t, err := r.toCommon()
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}

// AddTodo creates a new todo for today's effective date and returns the created row.
func (c *Client) AddTodo(text string) (common.Todo, error) {
	today := EffectiveDate(time.Now().Local())
	body := map[string]interface{}{
		"text":           text,
		"effective_date": today.Format("2006-01-02"),
	}
	headers := map[string]string{"Prefer": "return=representation"}
	data, err := c.doRequest("POST", "/rest/v1/todos", body, headers)
	if err != nil {
		return common.Todo{}, err
	}

	var rows []apiTodo
	if err := json.Unmarshal(data, &rows); err != nil {
		return common.Todo{}, fmt.Errorf("decoding created todo: %w", err)
	}
	if len(rows) == 0 {
		return common.Todo{}, fmt.Errorf("no todo returned after insert")
	}
	return rows[0].toCommon()
}

// ToggleTodo flips a todo's completed state. When marking complete it sets
// completed_at = now(); when un-marking it clears completed_at.
func (c *Client) ToggleTodo(id int, completed bool) error {
	body := map[string]interface{}{
		"completed": completed,
	}
	if completed {
		body["completed_at"] = time.Now().UTC().Format(time.RFC3339)
	} else {
		body["completed_at"] = nil
	}
	path := fmt.Sprintf("/rest/v1/todos?id=eq.%d", id)
	_, err := c.doRequest("PATCH", path, body, nil)
	return err
}

// EditTodo replaces a todo's text.
func (c *Client) EditTodo(id int, text string) error {
	body := map[string]interface{}{"text": text}
	path := fmt.Sprintf("/rest/v1/todos?id=eq.%d", id)
	_, err := c.doRequest("PATCH", path, body, nil)
	return err
}

// DeleteTodo removes a todo permanently.
func (c *Client) DeleteTodo(id int) error {
	path := fmt.Sprintf("/rest/v1/todos?id=eq.%d", id)
	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}
