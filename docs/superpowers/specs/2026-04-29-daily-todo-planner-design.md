# Daily Todo Planner — Design Spec

**Date:** 2026-04-29
**Status:** Approved, ready for implementation plan

## Goal

Add a daily to-do planner that resets every day at the 4 AM logical-day boundary already used by the rest of the app. Users add plain-text items, press space to check them off, and the list visually wipes at 4 AM. Old days remain queryable in the database so the user can navigate back through history with arrow keys, but they're read-only — past days cannot be edited, toggled, or added to.

The feature is a first-class tab on both the Go TUI and the SwiftUI iOS app, syncing through Supabase like every other table.

## Non-goals

- Carry-over of unfinished tasks. "Wipe" means the list resets; unfinished items don't pile up.
- Priorities, due times, categories, or tags. Items are plain text only.
- Habit linking, color coding, or "tap to start timer." Items have no relationship to habits beyond living in the same app.
- Reordering. Items show in insertion order — chronological.
- Widgets on iOS. Out of scope.
- A retention sweep / cleanup job. Past days stay in the DB indefinitely; we revisit if data volume ever matters.

## Behavior summary

- New tab labeled `Todo`, sitting 2nd in the tab order: `Timer | Todo | Stats | Habits | Log | Settings`.
- Default view shows today's list. `←` steps backward through days, `→` steps forward (capped at today), `0` jumps to today. Same navigation idiom as Stats tab's weekly chart.
- Today: full CRUD plus toggle. Past days: read-only.
- Day rollover happens at 4 AM via the existing `EffectiveDate` helper. No timer or background job; it's just a date filter.

## Architecture

### Module layout

A new `internal/todo/` package modeled on `internal/log/`:

- `internal/todo/model.go` — Bubble Tea sub-model: `New`, `Init`, `Update`, `View`, plus an `IsEditing()` predicate so `app.go` can gate the global `q`/`h`/`l` keys when an input is focused.
- `internal/api/client.go` gains methods: `ListTodos(date time.Time) ([]Todo, error)`, `AddTodo(text string) (Todo, error)`, `ToggleTodo(id int, completed bool) error`, `EditTodo(id int, text string) error`, `DeleteTodo(id int) error`.
- `internal/common/messages.go` gains: a `TodoTab` constant inserted at index 1, a `TodoRefreshMsg` struct, and a `Todo` struct (id, text, completed, effective date, created at, completed at).
- `internal/app/app.go` is updated:
  - `numTabs` 5 → 6.
  - `Model` gets a `todo todo.Model` field.
  - `New` constructs it; `Init`, `Update`, `View`, `updateActiveTab`, `updateAll`, `refreshTab` all wire it in.
  - The tab bar in `View` adds `{"Todo", common.TodoTab}` between Timer and Stats.

Day boundary: every reference to "today" or "this date" goes through `api.EffectiveDate(time.Now().Local())`. iOS uses the same logic from its existing helpers — a small shared util if not already factored.

### Database

Migration: `supabase/migrations/20260429000000_add_todos.sql`.

```sql
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

Notes:

- `effective_date` is a `DATE`, set by the client on insert using `api.EffectiveDate`. Server doesn't need TZ info — same model the rest of the schema uses for sessions.
- `text` capped at 200 chars; matches a glanceable list item and is enforced both client-side (input width / `CharLimit`) and server-side (CHECK constraint).
- `completed_at` is set to `now()` when toggling false → true (marking done), cleared (set to NULL) when toggling true → false (unmarking). Carries no UI weight today; included for future "history" surface.
- The compound index on `(user_id, effective_date desc)` makes "list my todos for date X" cheap and matches the dominant query pattern.
- Realtime subscription is added so iOS and TUI stay in sync across devices.

### "Wipe" mechanics

The wipe is virtual. There is no cron, no archive flag, no delete-on-rollover. At 4 AM, `api.EffectiveDate(time.Now())` starts returning a new date; the client's "today" filter naturally returns an empty list. Yesterday's data is unchanged in the table and visible via `←` navigation.

This simplifies everything: no background process, no failure mode where the wipe runs late or doesn't run at all, no race between realtime sync and a wipe action. The list "resets" the same way Sundays "happen" — by the calendar moving forward.

## TUI

### Layout

Header line:
- Today: `Todo` (title style) followed by a muted `Today`.
- Other days: `Todo` (title style) followed by a muted date string like `Mon Apr 28` and a hint `(press 0 for today)`.

List body:
- Each row: `  ▸ [ ] review chapter 3` for the cursor row, `    [ ] review chapter 3` for others (cursor `▸ ` matches Habits tab styling).
- Checked rows: `[✓]`, rendered in `MutedStyle` with a strike-through. Order is preserved — checked items don't sink to the bottom.
- Empty state today: `  No todos yet. Press [a] to add one.`
- Empty state past: `  No todos this day.`

Input row (when adding/editing): below the list, a single text input with placeholder `e.g. review chapter 3` and `CharLimit = 200`, mirroring the Log tab's input pattern.

Confirm dialog (when deleting): below the list, `  Delete "<text>"? [y]es [x] cancel` in `WarningStyle`, mirroring the Habits tab.

Help line at the bottom: shows the keymap for the current state (today / past / adding / editing / confirming).

### Keymap — today

| Key | Action |
|---|---|
| `j` / `↓` | Cursor down (wraps) |
| `k` / `↑` | Cursor up (wraps) |
| `a` | Enter add mode (input focused, empty) |
| `e` | Enter edit mode (input focused, prefilled with cursor row's text) |
| `space` | Toggle completed on cursor row |
| `d` | Enter confirm-delete mode for cursor row |
| `←` | Step to previous day (read-only view) |
| `→` | No-op on today (already the latest) |
| `0` | No-op on today |
| `enter` | Submit input (when in add/edit mode) |
| `esc` | Cancel input or confirm dialog |
| `y` | Confirm delete (in confirm-delete mode only) |
| `x` / `n` | Cancel delete (in confirm-delete mode only) |

### Keymap — past day

| Key | Action |
|---|---|
| `j` / `k` | Cursor up/down (cosmetic; no action available on a row) |
| `←` | Previous day |
| `→` | Next day (toward today) |
| `0` | Jump to today |

`a`, `space`, `e`, `d` are no-ops on past days. The help line shows them muted so the user understands the limitation.

### Edge cases

- Empty submit on add → input stays focused, no row created.
- Edit cleared to empty on submit → no-op (esc to cancel).
- `q` while input is focused → typed as text, doesn't quit. `app.go`'s `q` handler already skips when an editing sub-model says so via `IsEditing()`.
- Tab switching with input/confirm open → input/confirm is dropped, list reverts to read mode (matches Log tab).
- Adding a todo near 4 AM: the client computes `effective_date` at insert time. A todo created at 3:59 AM lands on yesterday; one created at 4:01 AM lands on today. This is consistent with how sessions are bucketed.
- Realtime delete from another device while cursor is on the deleted row: cursor clamps to last valid index, identical to how Log tab handles it on refresh.

### Sub-model state

```go
type viewMode int

const (
    modeNormal viewMode = iota
    modeAdd
    modeEdit
    modeConfirmDelete
)

type Model struct {
    client      *api.Client
    todos       []common.Todo
    cursor      int
    mode        viewMode
    input       textinput.Model
    editID      int
    viewingDate time.Time // result of api.EffectiveDate; equal to today on init
    width, height int
    lastErr     string
}
```

Helpers: `isToday()` returns whether `viewingDate` equals today, `IsEditing()` returns true for `modeAdd`/`modeEdit`.

## iOS

### View structure

`ios/tpom/tpom/TodoView.swift` — a `View` that takes a `DataService`.

Top: a header with the title `Todo` and a date pill. Pill text:
- `Today` when on today.
- `Mon Apr 28` when on a past day, with a small `chevron.right.circle` button beside it that returns to today.

Below the header: a small horizontal date strip with `chevron.left` and `chevron.right`. Right-chevron is disabled when on today.

Body: a `List` of todo rows. Each row:
- Leading: a `Circle` (uncompleted) or `checkmark.circle.fill` in `Theme.accent` (completed).
- Center: the text. Strikethrough + `Theme.muted` when completed.
- Tap on the row → toggles completion (the iOS equivalent of TUI `space`). Disabled on past days.
- Swipe-from-trailing reveals **Edit** and **Delete** actions. Delete uses `.confirmationDialog` to ask before destroying. Both are disabled on past days.

Bottom-right: a floating `+` button (FAB-style: `Button` with `Image(systemName: "plus.circle.fill")` styled in `Theme.accent`). Hidden on past days. Tapping it presents a sheet with a single `TextField` and Save/Cancel; same sheet is reused for edit (prefilled).

Empty states match the TUI text.

### Data flow

`DataService` (an `@Observable` already) gains:

- `var todos: [Todo] = []`
- `var viewingDate: Date` initialized to today's effective date via the existing iOS 4 AM helper (whatever it's called in `WidgetAPIClient.swift` / `DataService.swift`).
- `func loadTodos(for date: Date) async`
- `func addTodo(_ text: String) async`
- `func toggleTodo(id: Int) async`
- `func editTodo(id: Int, text: String) async`
- `func deleteTodo(id: Int) async`

`SupabaseClient` and `WidgetAPIClient` get matching REST methods, mirroring how sessions/habits are implemented.

`Models.swift` gains:
```swift
struct Todo: Identifiable, Codable, Equatable {
    let id: Int
    var text: String
    var completed: Bool
    let effectiveDate: Date  // date-only
    let createdAt: Date
    var completedAt: Date?
}
```

The 4 AM helper: inspect `WidgetAPIClient.swift`/`DataService.swift` for the existing implementation (sessions already bucket by 4 AM somewhere). Reuse it. If it's currently inlined, extract to a single shared helper as part of this work — small, in-scope cleanup.

### Realtime

`DataService.startRealtime()` already subscribes to other tables. Add `todos` to that subscription and refresh the current `viewingDate`'s list on any matching event.

### Widgets

No new widgets. The existing widgets (heatmap, summary, weekly chart, weekly habit) don't change.

### XcodeGen

`ios/tpom/project.yml` lists source paths. Adding `TodoView.swift` to `ios/tpom/tpom/` should be auto-included if the project uses path-based source resolution. If not, add it explicitly. Same for `Todo` model in `Shared/Models.swift` — already included.

## Error handling

Match the existing patterns in `habits/model.go` and `log/model.go`:

- TUI mutations (add/toggle/edit/delete) follow the existing **fetch-after-mutate** pattern: send the REST call, then re-list todos for the current `viewingDate`. On failure, set `m.lastErr = "Failed to <op> todo"` and let the next refresh retry naturally. No optimistic updates — keeps state simple and consistent with the rest of the codebase.
- iOS mutations call into `DataService`; on error, the `DataService` logs and surfaces nothing user-visible beyond the row not changing. Matches how the rest of the iOS app handles network failures.
- 401 / token refresh: handled by `api.Client.doRequest` already; nothing new needed.
- Realtime subscription failure: silently fall back to the existing tab-switch refresh path. The iOS app already does this; the TUI does too.
- Length validation: client rejects empty strings before calling the API; server CHECK rejects strings > 200 chars (defense in depth — should never fire in normal use).

## Testing

This codebase has minimal tests (only `internal/log/parser_test.go`), and the existing tabs ship without unit tests for their models. Don't introduce a new testing pattern as part of this work.

- **Manual TUI testing:** build `tpom`, exercise the keymap on today and a past day, verify the day rollover by manually adjusting the system clock past 4 AM and confirming today's list empties out and yesterday's `←` still shows the items. Verify cross-device sync by running TUI + iOS simultaneously.
- **Manual iOS testing:** in the simulator, exercise add/toggle/edit/delete, swipe actions, the date strip, and the empty states. Verify the round-trip with the TUI.
- **No new automated tests.** If we add a parser-style helper later (e.g., natural-language date input for past-day jumping), we'd add tests then; not needed now.

## Build sequence

This isn't the implementation plan — that comes next via writing-plans — but the obvious order is:

1. Schema migration + RLS + realtime.
2. Go API client methods.
3. Go `internal/todo/` sub-model.
4. Wire the new tab into `internal/app/app.go` and `internal/common/messages.go`.
5. Build, run, manually verify the TUI end-to-end.
6. iOS `Todo` model, `SupabaseClient` methods, `DataService` methods.
7. iOS `TodoView`, wire into `ContentView`.
8. iOS realtime subscription for `todos`.
9. Manually verify cross-device sync.
10. Update `README.md` keymap tables to include the Todo tab.

## Open questions

None at design time.
