# LazyTask

LazyTask is a terminal-first task manager inspired by LazyGit/LazyVim. It stores tasks in SQLite, offers a multi-pane TUI with quick navigation, and can optionally expose a simple read-only web view of your data.

## Features

- SQLite-backed tasks with full CRUD
- Multi-pane TUI: Pending, Recently Done, Tags, Highlighted, History
- Task history with per-field diffs
- Tag management with multi-select filtering
- Optional embedded web server for viewing tasks

## Getting Started

```bash
go run ./cmd/lazytask
```

Configuration is stored at `~/.config/lazytask/config.json`. A SQLite database file is created alongside it as `lazytask.db`.

### Web UI

```bash
go run ./cmd/lazytask --web
```

This starts a read-only web UI at `http://localhost:8080`.

## Keybindings

### Global

- `q` quit
- `r` reload
- `g` clear filters
- `h` refresh history
- `H` toggle history pane
- `?` help
- `tab` cycle panes
- `1-6` focus panes (Pending, Done, Tags, Highlighted, Eventually, History)

### Task Actions

- `a` add task
- `s` add subtask
- `e` edit task
- `d` delete task
- `c` toggle current
- `x` toggle done
- `v` toggle eventually

### Navigation

- `j/k` or arrow keys to move within list panes

### Search & Tags

- `/` search
- `space` toggle tag filter (in Tags pane)
- `ctrl+t` open tag picker (in task form)

## Form Editor

The task form is a single window showing all fields. Use:

- `tab` / `shift+tab` or `↑/↓` to move fields
- `enter` to save
- `esc` to cancel

## Development

```bash
go build ./...
```

## Notes

- History entries record full diffs for updates and full snapshots for create/delete.
- The web UI is intentionally minimal and read-only for now.
