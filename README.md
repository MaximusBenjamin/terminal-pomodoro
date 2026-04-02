# tpom - Terminal Pomodoro

A minimalist pomodoro timer for the terminal, built with Go. Lazygit-style TUI with habit tracking, heatmaps, and statistics.

![Go](https://img.shields.io/badge/Go-1.26-blue) ![License](https://img.shields.io/badge/License-MIT-green)

## Features

- **Timer** - Countdown with adjustable duration, pause/resume, overtime tracking
- **Habits** - Track time across categories (programming, mathematics, finance, etc.)
- **Stats** - Year-long heatmap, weekly per-category bar chart, today/week/all-time totals
- **Log** - View, add, edit, and delete sessions with natural language input
- **Dark theme** - Tokyo Night inspired color palette

## Install

```bash
# From source
go install github.com/MaximusBenjamin/terminal-pomodoro@latest

# Or clone and build
git clone https://github.com/MaximusBenjamin/terminal-pomodoro.git
cd terminal-pomodoro
go build -o tpom .
```

## Usage

```bash
tpom
```

### Navigation

| Key | Action |
|-----|--------|
| `h` / `l` | Switch tabs (left/right) |
| `q` | Quit |

### Timer

| Key | Action |
|-----|--------|
| `space` | Start / pause |
| `s` | Stop (prompts save/discard) |
| `r` | Reset (prompts save/discard) |
| `j` / `k` | Cycle habit |
| `+` / `-` | Adjust time by 5 min |

### Stats

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll |

### Habits

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `enter` | Select habit |
| `a` | Add habit |
| `d` | Delete habit |

### Log

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `a` | Add session (natural language) |
| `e` | Edit session |
| `d` | Delete session |

## Adding Sessions Manually

The log tab accepts natural language input:

```
30m math
2h programming
1.5h finance
1pm to 2pm programming
1:30pm - 2:30pm math yesterday
from 9am to 11am programming monday
45 minutes finance 25 march
30m math 01/04/2026
```

Habit names support prefix matching: `math` -> `mathematics`, `prog` -> `programming`.

## Data

All data is stored in SQLite at `~/.pomo/pomo.db`.

## License

MIT
