# LazyTask

LazyTask is a terminal-first task manager inspired by LazyGit/LazyVim. It stores tasks in SQLite, offers a multi-pane TUI with quick navigation, and can optionally expose a simple read-only web view of your data.

## Features

- SQLite-backed tasks with full CRUD
- Multi-pane TUI: Pending, Recently Done, Tags, Highlighted, History
- Task history with per-field diffs
- Tag management with multi-select filtering
- Optional embedded web server for viewing tasks

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap Joseda-hg/lazytask
brew install lazytask
```

### Winget (Windows)

```powershell
winget install Joseda-hg.LazyTask
```

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

### Versioning

Releases use `YY.MM.DD-Rev`, where `Rev` is the number of commits on that date. The embedded version is available via `--version`.

```bash
VERSION=$(./scripts/version.sh)
go build -ldflags "-X main.Version=$VERSION" ./cmd/lazytask
```

### DevContainer

Open the repo in VS Code and choose "Reopen in Container" to get Go, sqlc, and sqlite tooling preinstalled.

### Build

```bash
go build ./...
```

### Release Automation

Releases run automatically on pushes to `main` via `.github/workflows/release.yml`.

Required secrets:
- `HOMEBREW_TAP_TOKEN`: PAT with push access to `Joseda-hg/homebrew-lazytask`.
- `WINGET_GITHUB_TOKEN`: classic PAT with `public_repo` scope for winget PRs.

Winget requires a one-time bootstrap to create the `Joseda-hg.LazyTask` package in `microsoft/winget-pkgs`. After the first submission, the workflow uses `wingetcreate update` to open PRs on each release.

## Notes

- History entries record full diffs for updates and full snapshots for create/delete.
- The web UI is intentionally minimal and read-only for now.
