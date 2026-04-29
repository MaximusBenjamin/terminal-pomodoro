# Daily Todo Planner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Todo tab to both the Go TUI and the SwiftUI iOS app — a plain-text daily planner that visually wipes at the 4 AM day boundary, with past days kept in Supabase and accessible via arrow-key navigation. Past days are read-only.

**Architecture:** New `todos` table in Supabase with per-row `effective_date` (computed client-side). New `internal/todo/` Bubble Tea sub-model wired as the 2nd tab in `internal/app/app.go`. New `TodoView.swift` wired into `ContentView.swift`. Realtime sync via the existing `db-changes` channel. No background "wipe" job — the wipe is just a date filter.

**Tech Stack:** Go 1.26, Bubble Tea / Lip Gloss / Bubbles textinput, Supabase (PostgREST + Realtime), SwiftUI, supabase-swift 2.x, XcodeGen.

**Spec:** `docs/superpowers/specs/2026-04-29-daily-todo-planner-design.md` (read this first if any task is unclear).

**Conventions to follow:**
- This codebase has minimal automated tests; only `internal/log/parser_test.go` exists. Don't introduce new test patterns. Verify by `go build` + `go vet` + manual TUI run, and on iOS by `xcodegen` + `xcodebuild` + manual simulator run.
- Mutations follow the **fetch-after-mutate** pattern (see `internal/habits/model.go` `loadHabits` after add/delete).
- Errors surface via a `lastErr` string in the sub-model, rendered in `OvertimeStyle` (matches `habits/model.go:240–243`).
- Helpers go in the same file as the model unless reused (matches existing structure).
- Commit after each task. Use the commit-message style from `git log` (short, imperative).

---

## Task 1: Database migration

**Files:**
- Create: `supabase/migrations/20260429000000_add_todos.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- Daily todo planner. Each row belongs to a single effective day (4 AM
-- boundary, computed client-side). The "wipe" at end of day is virtual:
-- the client filters by effective_date = today, and the row stays in the
-- table indefinitely so past days remain navigable via the UI.

create table public.todos (
  id             bigint generated always as identity primary key,
  user_id        uuid references auth.users(id) on delete cascade not null default auth.uid(),
  text           text not null check (length(text) between 1 and 200),
  completed      boolean not null default false,
  effective_date date not null,
  created_at     timestamptz not null default now(),
  completed_at   timestamptz
);

create index todos_user_date on public.todos(user_id, effective_date desc);

alter table public.todos enable row level security;

create policy "Users manage own todos" on public.todos
  for all using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

alter publication supabase_realtime add table public.todos;
```

- [ ] **Step 2: Apply the migration via Supabase MCP**

Use the `mcp__supabase__apply_migration` tool with:
- `name`: `add_todos`
- `query`: the SQL from Step 1

If the MCP isn't available, run via `psql` against the Supabase project, or paste into the Supabase SQL editor. The migration must produce no errors.

- [ ] **Step 3: Verify the table and RLS policy**

Use `mcp__supabase__list_tables` (schema=`public`) and confirm `todos` appears with the columns above.

Use `mcp__supabase__execute_sql` with:
```sql
select policyname from pg_policies where tablename = 'todos';
```
Expected: a row with `Users manage own todos`.

Use `mcp__supabase__execute_sql` with:
```sql
select pubname from pg_publication_tables where tablename = 'todos';
```
Expected: a row with `supabase_realtime`.

- [ ] **Step 4: Commit**

```bash
git add supabase/migrations/20260429000000_add_todos.sql
git commit -m "Add todos table migration"
```

---

## Task 2: Common types — Todo struct, TodoTab const, TodoRefreshMsg

**Files:**
- Modify: `internal/common/messages.go`

- [ ] **Step 1: Add the Todo struct, refresh message, and tab constant**

Replace the entire contents of `internal/common/messages.go` with:

```go
package common

import "time"

// SessionEndMsg is sent when a timer session ends (user stops or timer completes).
type SessionEndMsg struct {
	HabitID         int
	PlannedMinutes  int
	ActualSeconds   int
	OvertimeSeconds int
	Completed       bool
}

// HabitSelectedMsg is sent when the user selects a habit.
type HabitSelectedMsg struct {
	ID    int
	Name  string
	Color string
}

// StatsRefreshMsg triggers a stats data reload.
type StatsRefreshMsg struct{}

// Habit represents a study habit.
type Habit struct {
	ID       int
	Name     string
	Color    string
	Archived bool
}

// LogRefreshMsg triggers a log data reload.
type LogRefreshMsg struct{}

// Todo represents a single daily-todo item.
type Todo struct {
	ID            int
	Text          string
	Completed     bool
	EffectiveDate time.Time // date-only (00:00:00 in local TZ); the day this todo belongs to
	CreatedAt     time.Time
	CompletedAt   *time.Time // nil when not completed
}

// TodoRefreshMsg triggers a todo data reload for the currently-viewed date.
type TodoRefreshMsg struct{}

// Tab identifiers
type Tab int

const (
	TimerTab Tab = iota
	TodoTab
	StatsTab
	HabitsTab
	LogTab
	SettingsTab
)
```

Note: inserting `TodoTab` at index 1 shifts `StatsTab`/`HabitsTab`/`LogTab`/`SettingsTab` by one. The codebase uses these constants by name only (verify via grep — no callsite compares to a literal int), so the shift is safe.

- [ ] **Step 2: Confirm no callsite depends on numerical tab values**

Run: `grep -RIn "Tab(\|common\.\(Timer\|Stats\|Habits\|Log\|Settings\)Tab" internal/ cmd/`

Expected: every reference uses the named constant (e.g. `common.HabitsTab`); no `common.Tab(2)` or `int(common.HabitsTab) ==` style comparisons.

- [ ] **Step 3: Build to verify**

Run: `go build ./...`
Expected: success (no compile errors).

- [ ] **Step 4: Commit**

```bash
git add internal/common/messages.go
git commit -m "Add Todo type and TodoTab constant"
```

---

## Task 3: Go API client — Todo CRUD methods

**Files:**
- Modify: `internal/api/client.go`

- [ ] **Step 1: Append the Todo API surface to `internal/api/client.go`**

Add the following code at the very end of the file (after the existing `SetLeeway` function, line ~712):

```go
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
```

- [ ] **Step 2: Build to verify**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/api/client.go
git commit -m "Add Todo CRUD methods to API client"
```

---

## Task 4: Go TUI sub-model — `internal/todo/model.go`

**Files:**
- Create: `internal/todo/model.go`

- [ ] **Step 1: Write the full sub-model**

Create `internal/todo/model.go` with:

```go
package todo

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/common"
)

type viewMode int

const (
	modeNormal viewMode = iota
	modeAdd
	modeEdit
	modeConfirmDelete
)

type todosLoadedMsg struct {
	todos []common.Todo
	err   string
}

// Model is the Bubble Tea sub-model for the todo view.
type Model struct {
	client      *api.Client
	todos       []common.Todo
	cursor      int
	mode        viewMode
	input       textinput.Model
	editID      int
	viewingDate time.Time // date-only; result of api.EffectiveDate
	width       int
	height      int
	lastErr     string
}

// New creates a new todo Model. Loads today's todos eagerly so the view has
// data before Init runs (matches the habits package pattern).
func New(c *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. review chapter 3"
	ti.CharLimit = 200
	ti.Width = 50

	m := Model{
		client:      c,
		input:       ti,
		viewingDate: api.EffectiveDate(time.Now().Local()),
	}

	if todos, err := c.ListTodos(m.viewingDate); err == nil {
		m.todos = todos
	}
	return m
}

// Init returns a command that loads today's todos.
func (m Model) Init() tea.Cmd {
	return m.loadTodos
}

func (m Model) loadTodos() tea.Msg {
	todos, err := m.client.ListTodos(m.viewingDate)
	if err != nil {
		return todosLoadedMsg{err: "Failed to load todos"}
	}
	return todosLoadedMsg{todos: todos}
}

// IsEditing returns true when the input is focused (add or edit).
func (m Model) IsEditing() bool {
	return m.mode == modeAdd || m.mode == modeEdit
}

// IsConfirming returns true when the delete confirm prompt is open.
func (m Model) IsConfirming() bool {
	return m.mode == modeConfirmDelete
}

func (m Model) isToday() bool {
	return m.viewingDate.Equal(api.EffectiveDate(time.Now().Local()))
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case todosLoadedMsg:
		if msg.err != "" {
			m.lastErr = msg.err
		} else {
			m.lastErr = ""
		}
		m.todos = msg.todos
		if m.cursor >= len(m.todos) {
			m.cursor = max(0, len(m.todos)-1)
		}
		return m, nil

	case common.TodoRefreshMsg:
		return m, m.loadTodos

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			return m.updateNormal(msg)
		case modeAdd:
			return m.updateInput(msg, false)
		case modeEdit:
			return m.updateInput(msg, true)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if len(m.todos) > 0 {
			m.cursor = (m.cursor + 1) % len(m.todos)
		}
		return m, nil

	case "k", "up":
		if len(m.todos) > 0 {
			m.cursor = (m.cursor - 1 + len(m.todos)) % len(m.todos)
		}
		return m, nil

	case "left":
		m.viewingDate = m.viewingDate.AddDate(0, 0, -1)
		m.cursor = 0
		return m, m.loadTodos

	case "right":
		today := api.EffectiveDate(time.Now().Local())
		if m.viewingDate.Before(today) {
			m.viewingDate = m.viewingDate.AddDate(0, 0, 1)
			m.cursor = 0
			return m, m.loadTodos
		}
		return m, nil

	case "0":
		today := api.EffectiveDate(time.Now().Local())
		if !m.viewingDate.Equal(today) {
			m.viewingDate = today
			m.cursor = 0
			return m, m.loadTodos
		}
		return m, nil
	}

	// Past-day view is read-only. The keys below only apply on today.
	if !m.isToday() {
		return m, nil
	}

	switch msg.String() {
	case "a":
		m.mode = modeAdd
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink

	case " ":
		if len(m.todos) > 0 {
			t := m.todos[m.cursor]
			newCompleted := !t.Completed
			id := t.ID
			return m, tea.Batch(
				func() tea.Msg {
					if err := m.client.ToggleTodo(id, newCompleted); err != nil {
						return todosLoadedMsg{todos: m.todos, err: "Failed to toggle"}
					}
					return nil
				},
				m.loadTodos,
			)
		}
		return m, nil

	case "e":
		if len(m.todos) > 0 {
			t := m.todos[m.cursor]
			m.mode = modeEdit
			m.editID = t.ID
			m.input.Reset()
			m.input.SetValue(t.Text)
			m.input.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "d":
		if len(m.todos) > 0 {
			m.mode = modeConfirmDelete
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg, isEdit bool) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			// No-op; stay in input mode.
			return m, nil
		}
		if isEdit {
			id := m.editID
			m.mode = modeNormal
			m.input.Reset()
			m.input.Blur()
			return m, tea.Batch(
				func() tea.Msg {
					if err := m.client.EditTodo(id, text); err != nil {
						return todosLoadedMsg{todos: m.todos, err: "Failed to edit"}
					}
					return nil
				},
				m.loadTodos,
			)
		}
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		return m, tea.Batch(
			func() tea.Msg {
				if _, err := m.client.AddTodo(text); err != nil {
					return todosLoadedMsg{todos: m.todos, err: "Failed to add"}
				}
				return nil
			},
			m.loadTodos,
		)

	case "esc":
		m.mode = modeNormal
		m.input.Reset()
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if len(m.todos) > 0 {
			id := m.todos[m.cursor].ID
			m.mode = modeNormal
			return m, tea.Batch(
				func() tea.Msg {
					if err := m.client.DeleteTodo(id); err != nil {
						return todosLoadedMsg{todos: m.todos, err: "Failed to delete"}
					}
					return nil
				},
				m.loadTodos,
			)
		}
		m.mode = modeNormal
		return m, nil

	case "x", "esc", "n":
		m.mode = modeNormal
		return m, nil
	}
	return m, nil
}

// View renders the todo view.
func (m Model) View() string {
	var b strings.Builder

	// Header.
	b.WriteString(common.TitleStyle.Render("Todo"))
	b.WriteString("  ")
	if m.isToday() {
		b.WriteString(common.MutedStyle.Render("Today"))
	} else {
		label := m.viewingDate.Format("Mon Jan 2")
		b.WriteString(common.MutedStyle.Render(label + "  (press 0 for today)"))
	}
	b.WriteString("\n\n")

	// List.
	if len(m.todos) == 0 {
		if m.isToday() {
			b.WriteString(common.MutedStyle.Render("  No todos yet. Press [a] to add one."))
		} else {
			b.WriteString(common.MutedStyle.Render("  No todos this day."))
		}
		b.WriteString("\n")
	} else {
		for i, t := range m.todos {
			cursor := "  "
			if i == m.cursor {
				cursor = common.AccentStyle.Render("▸ ")
			}
			box := "[ ]"
			if t.Completed {
				box = "[✓]"
			}
			textStyle := lipgloss.NewStyle()
			if t.Completed {
				textStyle = textStyle.Strikethrough(true).Foreground(lipgloss.Color("#565f89"))
			}
			line := fmt.Sprintf("%s%s %s", cursor, box, textStyle.Render(t.Text))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Input / confirm.
	if m.mode == modeAdd {
		b.WriteString("\n  ")
		b.WriteString(common.MutedStyle.Render("New: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else if m.mode == modeEdit {
		b.WriteString("\n  ")
		b.WriteString(common.MutedStyle.Render("Edit: "))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	} else if m.mode == modeConfirmDelete && len(m.todos) > 0 {
		t := m.todos[m.cursor]
		b.WriteString("\n")
		b.WriteString(common.WarningStyle.Render(fmt.Sprintf("  Delete \"%s\"?", t.Text)))
		b.WriteString("\n")
		b.WriteString("  " + helpEntry("y", "delete") + "  " + helpEntry("x", "cancel"))
		b.WriteString("\n")
	}

	if m.lastErr != "" {
		b.WriteString("\n")
		b.WriteString(common.OvertimeStyle.Render("  ⚠ " + m.lastErr))
		b.WriteString("\n")
	}

	// Help line.
	b.WriteString("\n")
	if m.mode == modeAdd || m.mode == modeEdit {
		b.WriteString("  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel"))
	} else if m.mode == modeConfirmDelete {
		// help already rendered above
	} else if m.isToday() {
		b.WriteString("  " +
			helpEntry("a", "add") + "  " +
			helpEntry("space", "toggle") + "  " +
			helpEntry("e", "edit") + "  " +
			helpEntry("d", "delete") + "  " +
			helpEntry("←", "prev day") + "  " +
			helpEntry("0", "today"))
	} else {
		b.WriteString("  " +
			helpEntry("←", "prev day") + "  " +
			helpEntry("→", "next day") + "  " +
			helpEntry("0", "today"))
	}

	return b.String()
}

func helpEntry(key, desc string) string {
	return common.HelpKeyStyle.Render("["+key+"]") + " " + common.HelpDescStyle.Render(desc)
}
```

- [ ] **Step 2: Build to verify**

Run: `go build ./...`
Expected: success.

Run: `go vet ./...`
Expected: no warnings.

- [ ] **Step 3: Commit**

```bash
git add internal/todo/model.go
git commit -m "Add todo TUI sub-model"
```

---

## Task 5: Wire the Todo tab into `app.go`

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Update `numTabs`, imports, struct, and constructor**

Change `internal/app/app.go:17`:
```go
const numTabs = 5
```
to:
```go
const numTabs = 6
```

Add to imports (between `"github.com/MaximusBenjamin/terminal-pomodoro/internal/timer"` and the closing paren):
```go
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/todo"
```

Add a `todo todo.Model` field to the `Model` struct so it reads (replacing `internal/app/app.go:20–31`):
```go
// Model is the top-level Bubble Tea model that orchestrates all views.
type Model struct {
	activeTab common.Tab
	timer     timer.Model
	stats     stats.Model
	habits    habits.Model
	log       logview.Model
	settings  settings.Model
	todo      todo.Model
	client    *api.Client
	width     int
	height    int
	ready     bool
}
```

Update `New` (replacing the body of `internal/app/app.go:34–56`) to:
```go
// New creates the top-level application model with all sub-models.
func New(c *api.Client) Model {
	h := habits.New(c)
	t := timer.New(c)
	se := settings.New(c)
	st := stats.New(c, se.Leeway())
	l := logview.New(c)
	td := todo.New(c)

	// Set the first habit as the default for the timer.
	sel := h.SelectedHabit()
	if sel.ID != 0 {
		t.SetHabit(sel)
	}

	return Model{
		activeTab: common.TimerTab,
		timer:     t,
		stats:     st,
		habits:    h,
		log:       l,
		settings:  se,
		todo:      td,
		client:    c,
	}
}
```

- [ ] **Step 2: Wire `Init`, `Update` size handling, `updateActiveTab`, `updateAll`, `refreshTab`**

Update `Init` (replacing `internal/app/app.go:59–67`):
```go
// Init initializes all sub-models.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.timer.Init(),
		m.stats.Init(),
		m.habits.Init(),
		m.log.Init(),
		m.settings.Init(),
		m.todo.Init(),
	)
}
```

In the `tea.WindowSizeMsg` branch of `Update` (after line 94, just before `return m, tea.Batch(cmds...)`), insert:
```go
		m.todo, cmd = m.todo.Update(inner)
		cmds = append(cmds, cmd)
```

In the `q` keypress handler (`internal/app/app.go:103–113`), add a guard so `q` is consumed by the todo input when active. Replace the existing `case "q":` block with:
```go
		case "q":
			if m.activeTab == common.HabitsTab {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			if m.activeTab == common.TodoTab && (m.todo.IsEditing() || m.todo.IsConfirming()) {
				break
			}
			if m.timer.IsConfirming() {
				break
			}
			return m, tea.Quit
```

Apply the same guarding to the `l` and `h` cases — replace lines 115–134 with:
```go
		case "l":
			if m.timer.IsConfirming() {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			if m.activeTab == common.TodoTab && (m.todo.IsEditing() || m.todo.IsConfirming()) {
				break
			}
			m.activeTab = (m.activeTab + 1) % numTabs
			return m, m.refreshTab()

		case "h":
			if m.timer.IsConfirming() {
				break
			}
			if m.activeTab == common.LogTab && m.log.IsEditing() {
				break
			}
			if m.activeTab == common.TodoTab && (m.todo.IsEditing() || m.todo.IsConfirming()) {
				break
			}
			m.activeTab = (m.activeTab + numTabs - 1) % numTabs
			return m, m.refreshTab()
```

Update `refreshTab` (replacing `internal/app/app.go:187–195`):
```go
// refreshTab sends a refresh message to the newly active tab so it picks up changes.
func (m Model) refreshTab() tea.Cmd {
	switch m.activeTab {
	case common.LogTab:
		return func() tea.Msg { return common.LogRefreshMsg{} }
	case common.StatsTab:
		return func() tea.Msg { return common.StatsRefreshMsg{} }
	case common.TodoTab:
		return func() tea.Msg { return common.TodoRefreshMsg{} }
	}
	return nil
}
```

Update `updateActiveTab` (replacing `internal/app/app.go:197–220`):
```go
func (m Model) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeTab {
	case common.TimerTab:
		m.timer, cmd = m.timer.Update(msg)
	case common.TodoTab:
		m.todo, cmd = m.todo.Update(msg)
	case common.StatsTab:
		m.stats, cmd = m.stats.Update(msg)
	case common.HabitsTab:
		m.habits, cmd = m.habits.Update(msg)
	case common.LogTab:
		m.log, cmd = m.log.Update(msg)
	case common.SettingsTab:
		prevLeeway := m.settings.Leeway()
		m.settings, cmd = m.settings.Update(msg)
		// Reload stats if leeway changed
		if m.settings.Leeway() != prevLeeway {
			m.stats.SetLeeway(m.settings.Leeway())
			var statsCmd tea.Cmd
			m.stats, statsCmd = m.stats.Update(common.StatsRefreshMsg{})
			cmd = tea.Batch(cmd, statsCmd)
		}
	}
	return m, cmd
}
```

Update `updateAll` (replacing `internal/app/app.go:222–238`) to forward to the todo sub-model too:
```go
func (m Model) updateAll(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.timer, cmd = m.timer.Update(msg)
	cmds = append(cmds, cmd)
	m.stats, cmd = m.stats.Update(msg)
	cmds = append(cmds, cmd)
	m.habits, cmd = m.habits.Update(msg)
	cmds = append(cmds, cmd)
	m.log, cmd = m.log.Update(msg)
	cmds = append(cmds, cmd)
	m.settings, cmd = m.settings.Update(msg)
	cmds = append(cmds, cmd)
	m.todo, cmd = m.todo.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
```

- [ ] **Step 3: Add the Todo tab to the tab bar and view switch**

Update the tab list and content switch in `View` (replacing `internal/app/app.go:247–287`):
```go
	// Build tab bar.
	tabs := []struct {
		label string
		tab   common.Tab
	}{
		{"Timer", common.TimerTab},
		{"Todo", common.TodoTab},
		{"Stats", common.StatsTab},
		{"Habits", common.HabitsTab},
		{"Log", common.LogTab},
		{"Settings", common.SettingsTab},
	}

	var tabParts []string
	for _, t := range tabs {
		if t.tab == m.activeTab {
			tabParts = append(tabParts, common.ActiveTabStyle.Render(t.label))
		} else {
			tabParts = append(tabParts, common.InactiveTabStyle.Render(t.label))
		}
	}
	nav := common.MutedStyle.Render("  ← h  l →")
	tabParts = append(tabParts, nav)
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabParts...)

	// Separator line.
	innerWidth := max(0, m.width-6)
	separator := common.MutedStyle.Render(strings.Repeat("─", innerWidth))

	// Active tab content.
	var content string
	switch m.activeTab {
	case common.TimerTab:
		content = m.timer.View()
	case common.TodoTab:
		content = m.todo.View()
	case common.StatsTab:
		content = m.stats.View()
	case common.HabitsTab:
		content = m.habits.View()
	case common.LogTab:
		content = m.log.View()
	case common.SettingsTab:
		content = m.settings.View()
	}
```

- [ ] **Step 4: Build and vet**

Run: `go build -o tpom .`
Expected: a fresh `tpom` binary appears in repo root with no compile errors.

Run: `go vet ./...`
Expected: no warnings.

- [ ] **Step 5: Manual smoke test of the TUI**

Run: `./tpom` (assumes user is logged in already; if not, the user runs `./tpom login` first).

Verify in the running TUI:
- The tab bar shows `Timer  Todo  Stats  Habits  Log  Settings` left-to-right.
- Pressing `l` from Timer moves to Todo.
- The Todo tab shows "Todo  Today" header and "No todos yet. Press [a] to add one."
- `a` opens an input. Type `test todo`, press `enter`. The row appears.
- `space` toggles the `[ ]` to `[✓]` and applies strike-through.
- `space` again toggles it back.
- `e` opens the editor pre-filled. Change text, `enter` saves.
- `d` opens the confirm prompt. `y` deletes; `x` cancels.
- `←` shows yesterday (likely "No todos this day."). Help line shows only navigation keys.
- `0` returns to today. Help line restores full keymap.
- `q` while input is open is typed (doesn't quit). `esc` closes input. `q` then quits.

If anything misbehaves, iterate on `internal/todo/model.go` until the smoke test passes.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go
git commit -m "Wire Todo tab into app"
```

---

## Task 6: iOS — Todo model in `Models.swift`

**Files:**
- Modify: `ios/tpom/Shared/Models.swift`

- [ ] **Step 1: Add `Todo` and `NewTodo` structs**

Append to `ios/tpom/Shared/Models.swift` (after the existing `NewSession` struct, before the stats display models around line 67):

```swift
// MARK: - Todo

struct Todo: Codable, Identifiable, Equatable {
    let id: Int
    let userId: UUID?
    let text: String
    let completed: Bool
    let effectiveDate: String   // YYYY-MM-DD
    let createdAt: String?
    let completedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case userId = "user_id"
        case text, completed
        case effectiveDate = "effective_date"
        case createdAt = "created_at"
        case completedAt = "completed_at"
    }
}

struct NewTodo: Codable {
    let text: String
    let effectiveDate: String  // YYYY-MM-DD

    enum CodingKeys: String, CodingKey {
        case text
        case effectiveDate = "effective_date"
    }
}
```

- [ ] **Step 2: Build the iOS project**

Run from repo root:
```bash
cd ios/tpom && xcodegen generate && xcodebuild -project tpom.xcodeproj -scheme tpom -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Expected: BUILD SUCCEEDED. (If iPhone 15 isn't available, substitute the name of any installed simulator: `xcrun simctl list devices available | grep iPhone`.)

- [ ] **Step 3: Commit**

```bash
git add ios/tpom/Shared/Models.swift ios/tpom/tpom.xcodeproj
git commit -m "iOS: add Todo model"
```

---

## Task 7: iOS — DataService + SupabaseClient methods

**Files:**
- Modify: `ios/tpom/tpom/DataService.swift`

- [ ] **Step 1: Add observable state for todos**

In `ios/tpom/tpom/DataService.swift`, find the existing observable properties at the top of the `DataService` class (around line 8–11) and replace them with:

```swift
    var habits: [Habit] = []
    var sessions: [Session] = []
    var todos: [Todo] = []
    var viewingTodoDate: Date = Date()
    var isLoading = false
    var error: String?
    private var realtimeTask: Task<Void, Never>?
```

The initial `viewingTodoDate` will be replaced by a proper today-effective-date value in the next step.

- [ ] **Step 2: Make the `effectiveDate` helper public-ish and add a date-only convenience**

The existing `effectiveDate(_:)` helper at the bottom of the file (around line 408) is `private`. Loosen it to `internal` (default access) and add a sibling helper that returns a `Date` (the start of the effective day in local time) so other files can pass a `Date` to `loadTodos(for:)`:

Replace the existing `private func effectiveDate(_ date: Date) -> DateComponents` and `private func parseDate(...)` block (around lines 408–420) with:

```swift
    func effectiveDate(_ date: Date) -> DateComponents {
        let shifted = Calendar.current.date(byAdding: .hour, value: -4, to: date)!
        return Calendar.current.dateComponents([.year, .month, .day], from: shifted)
    }

    /// Returns the effective day for `date` as a Date at midnight local time.
    /// Used by the Todo view to filter rows by `effective_date`.
    func effectiveDay(_ date: Date = Date()) -> Date {
        let comp = effectiveDate(date)
        return Calendar.current.date(from: comp) ?? date
    }

    private func parseDate(_ str: String) -> Date {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = f.date(from: str) { return d }
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: str) ?? Date()
    }
```

In the `init`/property line for `viewingTodoDate`, replace `var viewingTodoDate: Date = Date()` with the proper initializer. Since stored-property defaults can't reference instance methods directly, set it lazily in a constructor — add the following init right above the `// MARK: - Fetch` comment:

```swift
    init() {
        self.viewingTodoDate = Calendar.current.date(from: effectiveDate(Date())) ?? Date()
    }
```

- [ ] **Step 3: Add Todo CRUD + load methods to DataService**

Add a new section just below `// MARK: - Sessions` (around line 146 — after `updateSession`):

```swift
    // MARK: - Todos

    func loadTodos(for date: Date) async {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = .current
        let dateStr = formatter.string(from: date)
        do {
            todos = try await supabase.from("todos")
                .select()
                .eq("effective_date", value: dateStr)
                .order("created_at", ascending: true)
                .execute()
                .value
            viewingTodoDate = date
        } catch {
            self.error = friendlyError(error)
        }
    }

    func addTodo(text: String) async {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = .current
        let today = effectiveDay()
        let body = NewTodo(text: text, effectiveDate: formatter.string(from: today))
        do {
            try await supabase.from("todos")
                .insert(body)
                .execute()
            await loadTodos(for: viewingTodoDate)
        } catch {
            self.error = friendlyError(error)
        }
    }

    func toggleTodo(id: Int, completed: Bool) async {
        struct Patch: Encodable {
            let completed: Bool
            let completed_at: String?
        }
        let isoF = ISO8601DateFormatter()
        isoF.formatOptions = [.withInternetDateTime]
        let patch = Patch(
            completed: completed,
            completed_at: completed ? isoF.string(from: Date()) : nil
        )
        do {
            try await supabase.from("todos")
                .update(patch)
                .eq("id", value: id)
                .execute()
            await loadTodos(for: viewingTodoDate)
        } catch {
            self.error = friendlyError(error)
        }
    }

    func editTodo(id: Int, text: String) async {
        struct Patch: Encodable { let text: String }
        do {
            try await supabase.from("todos")
                .update(Patch(text: text))
                .eq("id", value: id)
                .execute()
            await loadTodos(for: viewingTodoDate)
        } catch {
            self.error = friendlyError(error)
        }
    }

    func deleteTodo(id: Int) async {
        do {
            try await supabase.from("todos")
                .delete()
                .eq("id", value: id)
                .execute()
            await loadTodos(for: viewingTodoDate)
        } catch {
            self.error = friendlyError(error)
        }
    }

    /// Whether the currently-viewed Todo day is today (and thus editable).
    var isTodoViewingToday: Bool {
        let today = effectiveDay()
        return Calendar.current.isDate(viewingTodoDate, inSameDayAs: today)
    }
```

- [ ] **Step 4: Have `fetchAll` also load today's todos**

In `fetchAll()` (around line 41–48), insert the todo load before the widget-snapshot write:

Replace:
```swift
    func fetchAll() async {
        isLoading = true
        await fetchHabits()
        await fetchSessions()
        await fetchLeeway()
        isLoading = false
        writeWidgetSnapshot()
    }
```
with:
```swift
    func fetchAll() async {
        isLoading = true
        await fetchHabits()
        await fetchSessions()
        await fetchLeeway()
        await loadTodos(for: effectiveDay())
        isLoading = false
        writeWidgetSnapshot()
    }
```

- [ ] **Step 5: Have `clearData` also clear todos**

In `clearData()` (around line 398–404), add `todos = []`:
```swift
    func clearData() {
        habits = []
        sessions = []
        todos = []
        error = nil
        stopRealtime()
        WidgetDataStore.clear()
    }
```

- [ ] **Step 6: Build**

```bash
cd ios/tpom && xcodegen generate && xcodebuild -project tpom.xcodeproj -scheme tpom -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 7: Commit**

```bash
git add ios/tpom/tpom/DataService.swift ios/tpom/tpom.xcodeproj
git commit -m "iOS: add Todo CRUD to DataService"
```

---

## Task 8: iOS — `TodoView.swift`

**Files:**
- Create: `ios/tpom/tpom/TodoView.swift`

- [ ] **Step 1: Write the SwiftUI view**

Create `ios/tpom/tpom/TodoView.swift` with:

```swift
import SwiftUI

struct TodoView: View {
    var dataService: DataService

    @State private var showAddSheet = false
    @State private var editingTodo: Todo?
    @State private var deleteCandidate: Todo?

    var body: some View {
        ZStack {
            Theme.bg.ignoresSafeArea()

            VStack(spacing: 0) {
                header
                dateStrip
                Divider().background(Theme.border)
                listBody
            }

            // Floating add button (today only).
            if dataService.isTodoViewingToday {
                VStack {
                    Spacer()
                    HStack {
                        Spacer()
                        Button {
                            editingTodo = nil
                            showAddSheet = true
                        } label: {
                            Image(systemName: "plus.circle.fill")
                                .font(.system(size: 52))
                                .foregroundStyle(Theme.accent)
                                .background(Circle().fill(Theme.bg))
                        }
                        .padding(.trailing, 20)
                        .padding(.bottom, 24)
                    }
                }
            }
        }
        .sheet(isPresented: $showAddSheet) {
            TodoFormSheet(
                dataService: dataService,
                editing: editingTodo
            )
        }
        .confirmationDialog(
            "Delete this todo?",
            isPresented: Binding(
                get: { deleteCandidate != nil },
                set: { if !$0 { deleteCandidate = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) {
                if let todo = deleteCandidate {
                    Task { await dataService.deleteTodo(id: todo.id) }
                }
                deleteCandidate = nil
            }
            Button("Cancel", role: .cancel) {
                deleteCandidate = nil
            }
        } message: {
            Text(deleteCandidate?.text ?? "")
        }
    }

    // MARK: - Header

    private var header: some View {
        HStack {
            Text("Todo")
                .font(.title2.bold())
                .foregroundStyle(Theme.accent)
            Spacer()
        }
        .padding(.horizontal)
        .padding(.top, 8)
        .padding(.bottom, 8)
    }

    // MARK: - Date strip

    private var dateStrip: some View {
        HStack(spacing: 12) {
            Button {
                Task { await stepDay(by: -1) }
            } label: {
                Image(systemName: "chevron.left")
                    .foregroundStyle(Theme.accent)
            }

            Text(dateLabel)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(Theme.fg)
                .frame(minWidth: 120)

            Button {
                Task { await stepDay(by: 1) }
            } label: {
                Image(systemName: "chevron.right")
                    .foregroundStyle(canStepForward ? Theme.accent : Theme.muted)
            }
            .disabled(!canStepForward)

            Spacer()

            if !dataService.isTodoViewingToday {
                Button("Today") {
                    Task { await dataService.loadTodos(for: dataService.effectiveDay()) }
                }
                .font(.caption.weight(.medium))
                .foregroundStyle(Theme.accent)
            }
        }
        .padding(.horizontal)
        .padding(.bottom, 8)
    }

    private var dateLabel: String {
        if dataService.isTodoViewingToday {
            return "Today"
        }
        let f = DateFormatter()
        f.dateFormat = "EEE MMM d"
        return f.string(from: dataService.viewingTodoDate)
    }

    private var canStepForward: Bool {
        let today = dataService.effectiveDay()
        return dataService.viewingTodoDate < today
    }

    private func stepDay(by days: Int) async {
        let cal = Calendar.current
        guard let next = cal.date(byAdding: .day, value: days, to: dataService.viewingTodoDate) else { return }
        let today = dataService.effectiveDay()
        let target = next > today ? today : next
        await dataService.loadTodos(for: target)
    }

    // MARK: - List

    @ViewBuilder
    private var listBody: some View {
        if dataService.todos.isEmpty {
            Spacer()
            VStack(spacing: 12) {
                Image(systemName: "checklist")
                    .font(.system(size: 40))
                    .foregroundStyle(Theme.muted)
                Text(dataService.isTodoViewingToday
                     ? "No todos yet"
                     : "No todos this day")
                    .foregroundStyle(Theme.muted)
                if dataService.isTodoViewingToday {
                    Text("Tap + to add one")
                        .font(.caption)
                        .foregroundStyle(Theme.muted)
                }
            }
            Spacer()
        } else {
            List {
                ForEach(dataService.todos) { todo in
                    todoRow(todo)
                        .listRowBackground(Theme.border.opacity(0.15))
                        .contentShape(Rectangle())
                        .onTapGesture {
                            guard dataService.isTodoViewingToday else { return }
                            Task { await dataService.toggleTodo(id: todo.id, completed: !todo.completed) }
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if dataService.isTodoViewingToday {
                                Button(role: .destructive) {
                                    deleteCandidate = todo
                                } label: {
                                    Label("Delete", systemImage: "trash")
                                }
                                Button {
                                    editingTodo = todo
                                    showAddSheet = true
                                } label: {
                                    Label("Edit", systemImage: "pencil")
                                }
                                .tint(Theme.accent)
                            }
                        }
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
        }
    }

    @ViewBuilder
    private func todoRow(_ todo: Todo) -> some View {
        HStack(spacing: 12) {
            Image(systemName: todo.completed ? "checkmark.circle.fill" : "circle")
                .font(.title3)
                .foregroundStyle(todo.completed ? Theme.accent : Theme.muted)

            Text(todo.text)
                .strikethrough(todo.completed)
                .foregroundStyle(todo.completed ? Theme.muted : Theme.fg)

            Spacer()
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Add/Edit sheet

private struct TodoFormSheet: View {
    var dataService: DataService
    var editing: Todo?

    @Environment(\.dismiss) private var dismiss
    @State private var text: String = ""
    @State private var isSaving = false

    private var isEditing: Bool { editing != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("e.g. review chapter 3", text: $text, axis: .vertical)
                        .foregroundStyle(Theme.fg)
                        .lineLimit(1...4)
                } header: {
                    Text(isEditing ? "Edit todo" : "New todo")
                        .foregroundStyle(Theme.muted)
                }
                .listRowBackground(Theme.border.opacity(0.2))
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle(isEditing ? "Edit Todo" : "Add Todo")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(Theme.muted)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") { save() }
                        .fontWeight(.semibold)
                        .foregroundStyle(Theme.accent)
                        .disabled(isSaving || trimmed.isEmpty)
                }
            }
            .onAppear {
                if let editing { text = editing.text }
            }
        }
        .presentationBackground(Theme.bg)
        .presentationDetents([.medium])
    }

    private var trimmed: String {
        text.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func save() {
        let value = trimmed
        guard !value.isEmpty else { return }
        isSaving = true
        Task {
            if let editing {
                await dataService.editTodo(id: editing.id, text: value)
            } else {
                await dataService.addTodo(text: value)
            }
            dismiss()
        }
    }
}
```

- [ ] **Step 2: Build**

```bash
cd ios/tpom && xcodegen generate && xcodebuild -project tpom.xcodeproj -scheme tpom -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 3: Commit**

```bash
git add ios/tpom/tpom/TodoView.swift ios/tpom/tpom.xcodeproj
git commit -m "iOS: add TodoView"
```

---

## Task 9: Wire `TodoView` into `ContentView`

**Files:**
- Modify: `ios/tpom/tpom/ContentView.swift`

- [ ] **Step 1: Insert the Todo tab between Timer and Stats**

Replace the body of `ContentView`'s `TabView` (lines 8–29) with:

```swift
        TabView(selection: $selectedTab) {
            TimerView(dataService: dataService)
                .tabItem { Label("Timer", systemImage: "timer") }
                .tag(0)

            TodoView(dataService: dataService)
                .tabItem { Label("Todo", systemImage: "checklist") }
                .tag(1)

            StatsView(dataService: dataService)
                .tabItem { Label("Stats", systemImage: "chart.bar") }
                .tag(2)

            HabitsListView(dataService: dataService)
                .tabItem { Label("Habits", systemImage: "list.bullet") }
                .tag(3)

            LogView(dataService: dataService)
                .tabItem { Label("Log", systemImage: "clock.arrow.circlepath") }
                .tag(4)

            SettingsView(dataService: dataService)
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(5)
        }
```

- [ ] **Step 2: Refresh today's todos when the app returns to active**

In the `.onChange(of: scenePhase)` handler at the bottom of `ContentView` (around line 35–44), add a `loadTodos` call alongside the existing `fetchAll`:

Replace:
```swift
        .onChange(of: scenePhase) { _, newPhase in
            Task {
                if newPhase == .active {
                    await dataService.fetchAll()
                    await dataService.startRealtime()
                } else if newPhase == .background {
                    dataService.stopRealtime()
                }
            }
        }
```
with:
```swift
        .onChange(of: scenePhase) { _, newPhase in
            Task {
                if newPhase == .active {
                    await dataService.fetchAll()
                    // If a previous session was on a past day and we passed 4 AM,
                    // returning to active should snap viewing back to today.
                    if !dataService.isTodoViewingToday && Calendar.current.isDateInToday(dataService.viewingTodoDate) == false {
                        await dataService.loadTodos(for: dataService.effectiveDay())
                    }
                    await dataService.startRealtime()
                } else if newPhase == .background {
                    dataService.stopRealtime()
                }
            }
        }
```

- [ ] **Step 3: Build and run in the simulator**

```bash
cd ios/tpom && xcodegen generate && xcodebuild -project tpom.xcodeproj -scheme tpom -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Open the project in Xcode (`open ios/tpom/tpom.xcodeproj`), select an iPhone simulator, and run. Verify in the simulator:
- Tab bar order: Timer / Todo / Stats / Habits / Log / Settings.
- Empty state on Todo tab.
- Tap `+`, the sheet opens; type a todo, save; row appears.
- Tap the row → completes (filled circle, strikethrough).
- Swipe-left → Edit and Delete actions appear; Delete prompts for confirmation.
- Tap left chevron → past-day view; rows appear non-interactive (no tap toggle, no swipe). FAB hidden. "Today" pill appears; tap returns to today.

If anything misbehaves, iterate on `TodoView.swift` until the smoke test passes.

- [ ] **Step 4: Commit**

```bash
git add ios/tpom/tpom/ContentView.swift ios/tpom/tpom.xcodeproj
git commit -m "iOS: wire Todo tab into ContentView"
```

---

## Task 10: iOS realtime subscription for `todos`

**Files:**
- Modify: `ios/tpom/tpom/DataService.swift`

- [ ] **Step 1: Subscribe to `todos` table changes**

Find the existing `startRealtime` function (around lines 150–177). Replace it with:

```swift
    func startRealtime() async {
        // Don't start twice
        guard realtimeTask == nil else { return }

        let channel = supabase.realtimeV2.channel("db-changes")

        let sessionChanges = channel.postgresChange(AnyAction.self, table: "sessions")
        let habitChanges = channel.postgresChange(AnyAction.self, table: "habits")
        let todoChanges = channel.postgresChange(AnyAction.self, table: "todos")

        await channel.subscribe()

        realtimeTask = Task {
            await withTaskGroup(of: Void.self) { group in
                group.addTask {
                    for await _ in sessionChanges {
                        await self.fetchSessions()
                        self.writeWidgetSnapshot()
                    }
                }
                group.addTask {
                    for await _ in habitChanges {
                        await self.fetchHabits()
                        self.writeWidgetSnapshot()
                    }
                }
                group.addTask {
                    for await _ in todoChanges {
                        await self.loadTodos(for: self.viewingTodoDate)
                    }
                }
            }
        }
    }
```

- [ ] **Step 2: Build**

```bash
cd ios/tpom && xcodegen generate && xcodebuild -project tpom.xcodeproj -scheme tpom -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15' build
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 3: Commit**

```bash
git add ios/tpom/tpom/DataService.swift ios/tpom/tpom.xcodeproj
git commit -m "iOS: subscribe to todos realtime channel"
```

---

## Task 11: Cross-device verification + README update

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update the README keymap**

Open `README.md`. Just below the `### Timer` section (around line 80), insert a new section:

```markdown
### Todo

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `a` | Add todo |
| `space` | Toggle done |
| `e` | Edit todo |
| `d` | Delete todo (asks to confirm) |
| `←` / `→` | Previous / next day |
| `0` | Today |

Past days are read-only.
```

Also extend the "Features" bullet list at line 17 by inserting:

```markdown
- **Todo** - Daily plain-text todo list that resets at the 4 AM day boundary. Past days are kept and navigable read-only.
```

- [ ] **Step 2: Run cross-device sync verification**

Build a fresh Go binary and start it, then run the iOS app in a simulator (or your phone) signed into the same account.

```bash
go build -o tpom .
./tpom
```

Cross-device checklist:
- Add a todo in the TUI → it appears in iOS within ~1s (realtime).
- Add a todo in iOS → it appears in the TUI on next refresh (or instantly if you tab away and back).
- Toggle a todo in either → the other updates within ~1s.
- Delete in either → the other reflects the deletion.
- Edit text in either → the other updates the row.
- Navigate to yesterday on both → both show the same (or both empty) historical list.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "Document Todo tab in README"
```

---

## Done

The feature is complete when:
- The `todos` table exists in Supabase with RLS + realtime.
- The TUI shows a `Todo` tab between Timer and Stats with the keymap from Task 5 Step 5.
- The iOS app shows a `Todo` tab with the layout from Task 9 Step 3.
- Adding/editing/toggling/deleting in one client appears in the other within ~1s.
- Past days show as read-only on both clients.
- The "wipe at 4 AM" behavior is verified by either (a) waiting through 4 AM, or (b) manually setting the system clock past 4 AM and confirming today's list empties while yesterday's remains accessible via `←`.
