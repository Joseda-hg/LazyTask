# AGENTS

## Repository Overview

LazyTask is a Go CLI + TUI application that stores tasks in SQLite and optionally exposes a read-only web UI. The TUI is built with `github.com/jesseduffield/gocui`. The data layer uses sqlc with schema + queries under `internal/db/`.

There are no Cursor/Copilot rules present in this repo.

## Build / Lint / Test Commands

### Build

- Build all packages:
  - `go build ./...`
- Build the main binary:
  - `go build ./cmd/lazytask`

### Run

- Run the TUI:
  - `go run ./cmd/lazytask`
- Run with web UI:
  - `go run ./cmd/lazytask --web`
- Run web only:
  - `go run ./cmd/lazytask --web --web-only`

### SQLC (code generation)

- Generate SQLC output:
  - `sqlc generate`

### Format

- Format all Go code:
  - `gofmt -w ./...`

### Lint

- No linter is configured in this repo.
- If you add one, document it here and prefer `golangci-lint`.

### Tests

- There are currently no tests.
- Standard commands if tests are added:
  - Run all tests: `go test ./...`
  - Run a single package: `go test ./internal/tui`
  - Run a single test by name:
    - `go test ./internal/tui -run TestName`

## Code Style Guidelines

### Go Formatting

- Always run `gofmt` on modified Go files.
- Keep lines readable (prefer shorter lines, but gofmt is the primary rule).

### Imports

- Use Go’s standard import grouping order:
  1) Standard library
  2) Third-party packages
  3) Local modules
- Keep a blank line between each group.
- Avoid unused imports; prefer explicit imports over dot/blank except for drivers.

### Naming

- Follow Go conventions:
  - PascalCase for exported types/functions.
  - camelCase for unexported identifiers.
  - Short, descriptive names for local variables.
- Avoid one-letter names except for simple loop indices.

### Types

- Prefer explicit types in public APIs.
- Use `int64` for database IDs and persisted numeric fields.
- Use pointer types for optional values (e.g., `*time.Time` for nullable dates).

### Error Handling

- Return errors early with context.
- Use `fmt.Errorf("context: %w", err)` for wrapping.
- Avoid swallowing errors; log or return them.
- Ensure DB operations handle and return errors properly.

### SQLite / SQLC

- Database schema is in `internal/db/schema.sql`.
- Queries live in `internal/db/queries.sql`.
- Regenerate SQLC output after changing schema/queries.

### TUI (gocui)

- Views are created in `internal/tui/tui.go`.
- Keep view titles and keybindings in sync with the footer/help text.
- Use `applyViewStyle` for consistent pane appearance.
- Input overlays (search/help/form/tag create) should always capture focus.
- When a new status is added, update `cycleStatus()` and the Eventually pane logic.

### CLI / Config

- Config lives in `~/.config/lazytask/config.json`.
- CLI flags are parsed in `cmd/lazytask/main.go`.

### Web UI

- Web server is intentionally read-only.
- Templates live in `internal/web/templates`.

## Common Pitfalls

- Ensure `sqlc generate` is run after SQL changes.
- Avoid reusing Bubbletea-specific patterns in the gocui TUI.
- Keep focus transitions in sync (`SetCurrentView`) to prevent “unknown view” errors.
- When changing keybindings, update `helpText()` and the footer hint.

## File Layout

- `cmd/lazytask/main.go`: CLI entrypoint
- `internal/db/`: schema + store + sqlc output
- `internal/tui/`: gocui TUI
- `internal/web/`: embedded web UI
- `internal/config/`: config loader/saver

## Style Do/Don’t

- **Do** use `formatTaskSummary` for list display.
- **Do** update history entries when task fields change.
- **Do** keep the history view formatting consistent.
- **Don’t** add inline comments unless requested.
- **Don’t** introduce new dependencies without a clear need.
- **Don’t** mutate shared state without re-rendering affected views.

## Maintenance Notes

- The tag pane supports multi-select filters; tasks must include all selected tags.
- The form’s Tags and Status fields are selection-based; they do not accept free typing.
- The tag create dialog is launched from the Tags pane.
- History entries store diff-style updates and full snapshots for create/delete.
- The Eventually pane lists tasks with status `eventually`.
