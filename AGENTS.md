# Agent Guidelines for lazylogcat

TUI application for viewing Android logcat logs, built with Go 1.25+ and Bubble Tea v2. Requires `adb` in PATH for most commands; exceptions include `lazylogcat web --demo` and `lazylogcat logs parse` on stdin **without** a package filter (package filter needs adb for PID resolution).

## Build, Test & Run

```bash
# Build
go build -v ./...              # Build all packages
go build -o lazylogcat .       # Build binary

# Run
go run .                       # Launch TUI (same as go run main.go)
go run . --debug               # Debug logging to .lazylogcat.log

# Test
go test -v ./...                                           # All tests
go test -v -race ./...                                     # With race detector (CI default)
go test -v -coverprofile=coverage.out ./...                # With coverage
go test -v ./internal/config                               # Single package
go test -v -run TestDefaultConfig ./internal/config        # Single test
go test -v -run TestAppend/ExceedCapacity ./internal/util  # Single subtest
go tool cover -func=coverage.out                           # Coverage report

# Lint & Verify
go fmt ./...       # Format
go vet ./...       # Static analysis
go mod tidy        # Clean go.mod
go mod verify      # Verify deps
```

### Frontend (web-ui) — requires `bun`

```bash
(cd web-ui && bun install)        # Install JS dependencies (one-time setup)
go generate ./internal/web/...    # Build frontend into internal/web/static/
bun run --cwd web-ui dev          # Dev server with HMR (proxies /api, /ws to :8321)
```

## Architecture

**TUI entry flow**: `main.go` → `cmd/root.go` (Cobra CLI) → `internal/app/app.go` (`PreLaunchChecks`, `LaunchTUI`) → `internal/tui/mainui/tui.go`

**Web entry flow**: `main.go` → `cmd/web.go` (Cobra CLI) → `internal/web/server.go` (HTTP + WebSocket server, log streaming, embedded static UI)

Uses the **Elm Architecture** (Model-View-Update) via Bubble Tea. `MainModel` is the root state machine coordinating views via `sessionState` iota enum. Custom messages (`tea.Msg`) drive all state transitions; `tea.Cmd` handles async operations.

## CLI overview

| Command | Role |
|---------|------|
| _(no subcommand)_ | Interactive TUI (default) |
| `web` | Experimental browser UI + API |
| `devices` | List devices (`--output json` \| `text`) |
| `config show` | Print merged config as JSON |
| `logs dump` | Stream/collect filtered logcat to stdout |
| `logs parse` | Parse log lines from stdin |
| `skill install` | Copy embedded skill into agent dirs (`--agent`, `--user` \| `--project`) |
| `version` | Print version |

**Persistent flags** (root): `--debug`; `--config` / `--config-local` override project and local config file paths (same merge rules as TUI).

**Filter flags** on default run and `web`: `--pkg`, `--tag`, `--text` (contain match; override config when passed).

## Project Structure

```
├── cmd/                    # Cobra: root, web, version, skill, devices, config, logs
├── skills/                  # Bundled agent skill(s), embedded (`//go:embed`)
│   └── lazylogcat/         # SKILL.md; `skills` package exposes bytes for installers
├── internal/
│   ├── app/                # Pre-launch checks, logging setup, TUI launch
│   ├── config/             # Layered config: defaults → global → project → local
│   ├── model/              # Device, Filter, LogLine, Command*, OutputPrefs, Columns, Size, Level, TextFilter*, Shortcut, …
│   ├── skillinstall/      # Paths + copy for `lazylogcat skill install`
│   ├── tui/
│   │   ├── commandui/     # Command palette, dialogs, inputs, table
│   │   ├── commonui/      # Shared dialog shell
│   │   ├── helpui/        # Shortcut help overlay
│   │   ├── logcatui/      # Log stream view, search, visual mode
│   │   ├── mainui/        # Root model and state machine
│   │   └── theme/         # ANSI-16 styles
│   ├── util/               # ADB, logcat parse/read, ring buffer, clipboard, editor, mappers
│   └── web/                # HTTP + WebSocket server (experimental web UI backend)
├── web-ui/                 # React + Vite frontend → internal/web/static/
├── config.schema.json      # JSON Schema for config files
└── main.go                 # Entry → cmd.Execute()
```

## Web UI (experimental)

> **Warning:** The web experience is experimental and may change or break without notice.

### Command

```
lazylogcat web [flags]
  --port int     Port to listen on (default 8321)
  --demo         Run with a fake device and synthetic log lines (no adb required)
  --pkg string   Filter by package name (contains match, overrides config)
  --tag string   Filter by log tag (contains match, overrides config)
  --text string  Filter by log text (contains match, overrides config)
```

`--debug` is a **root** persistent flag (same as TUI). Web command inherits it via `PreRunE`.

Starts an HTTP server, opens the browser automatically, and waits for SIGINT/SIGTERM to shut down.

### HTTP Endpoints

Routing uses Go 1.22+ `ServeMux` patterns (`METHOD /path`).

| Endpoint | Description |
|----------|-------------|
| `GET /api/devices` | Returns connected ADB devices as JSON (demo: single fake device) |
| `GET /api/config` | Returns resolved app config as JSON |
| `GET /ws` | WebSocket for real-time log streaming |
| `GET /` | Serves embedded static frontend |

### WebSocket Protocol

Client→server message types: `connect` (payload: `deviceId`, `filter`), `disconnect`, `updateFilter` (payload: `filter`), `listDevices`

Server→client message types: `lines`, `connected`, `disconnected`, `devices`, `error`, `clearLines` (sent after `updateFilter` so the client can drop buffered lines)

Each WebSocket connection gets its own `Session` with a `LogcatReader` (or demo reader) and a `RingBuffer` with capacity `10000`. Log lines are drained every 50ms and sent in batches.

### Frontend Stack

- **Framework:** React 19 + TypeScript
- **Build tool:** Vite 6 via `bun`
- **Styling:** Emotion (`@emotion/react`, `@emotion/styled`); Roboto via `@fontsource/roboto`
- **Output:** `internal/web/static/` — embedded into the Go binary via `//go:embed static/*`
- **UI primitives:** MUI 7 (`@mui/material`, `@mui/icons-material`)
- **Virtualization:** `@tanstack/react-virtual` v3 — windowed log list
- **Dev proxy:** Vite dev server (`bun run dev`) proxies `/api` and `/ws` to `localhost:8321`

### Key Types (`internal/web` and friends)

- **`LogcatReader`** (`internal/util/logcat_reader.go`) — per-session adb logcat reader with internal goroutine. Methods: `Connect(deviceId, filter)`, `Disconnect()`, `Drain()`, `UpdateFilter()`, `UpdatePIDSet()`, `WaitForDone()`, `IsConnected()`, `Err()`.
- **`RingBuffer`** (`internal/util/buffer.go`) — thread-safe ring buffer of `model.LogLine` with configurable capacity (`NewRingBuffer(capacity int)`). Methods: `Append()`, `Recent(n)`, `All()`, `Size()`, `Clear()`.
- **`OutputPrefs`** (`internal/model/output_prefs.go`) — controls rendering column visibility and soft-wrap for the web output format.
- **`Size`** (`internal/model/terminal.go`) — terminal/viewport dimensions (`Width`, `Height int`).

## Code Style

### Imports

Three groups separated by blank lines: stdlib, external, internal. Always alias bubbletea as `tea`:

```go
import (
    "fmt"
    "log/slog"

    "charm.land/bubbles/v2/viewport"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"

    "github.com/parfenovvs/lazylogcat/internal/model"
    "github.com/parfenovvs/lazylogcat/internal/tui"
)
```

### Naming

- **Packages**: lowercase, single word or compound (`config`, `commandui`, `logcatui`)
- **Files**: snake_case (`logcat_view.go`, `buffer_test.go`)
- **Types**: PascalCase exported (`RingBuffer`, `MainModel`), camelCase unexported (`sessionState`, `dialogState`)
- **Functions**: PascalCase exported (`NewRingBuffer`), camelCase unexported (`readNext`, `buildRows`)
- **Variables**: camelCase (`deviceId`, `logcatCmd`)
- **Constants**: PascalCase for exported and iota enums (`LvlV`, `CommandPackage`), camelCase for unexported (`batchTimeout`, `maxLogLines`)
- **Sentinel errors**: `Err` prefix, PascalCase (`ErrAdbNotFound`, `ErrFailedToGetDevices`)

### Error Handling

```go
// Package-level sentinel errors
var (
    ErrAdbNotFound        = fmt.Errorf("adb not found")
    ErrFailedToGetDevices = fmt.Errorf("failed to get connected devices")
)

// Wrap with context using %w
return fmt.Errorf("could not open log file: %w", err)

// Double-wrap sentinel + original error
return fmt.Errorf("%w: %w", ErrFailedToGetStdoutPipe, err)

// Aggregate non-fatal errors with errors.Join
return cfg, errors.Join(errs...)
```

### Logging

Use `log/slog` with structured key-value pairs. Use `"error"` as the key for error values:

```go
slog.Debug("Configuration loaded", "config", c.String())
slog.Warn("Failed to load config file, skipping", "path", path, "error", err)
```

### Concurrency

Use `sync.RWMutex` with immediate `defer` unlock. TUI async work goes through Bubble Tea's `tea.Cmd` mechanism, not ad-hoc goroutines from views. `internal/web` and `internal/util.LogcatReader` use background goroutines managed with `context` cancel / `WaitForDone`.

## Bubble Tea Patterns

- Only `MainModel` returns `tea.Model` from `Update`; sub-models return their concrete type (e.g., `(LogcatViewModel, tea.Cmd)`) to avoid type assertions
- Messages are structs (even empty ones) with `Msg` suffix; navigation signals use `Cmd` suffix
- Unexported messages (`logcatMsg`) stay within a component; exported ones (`DeviceSelectedMsg`) cross boundaries
- Views use iota-based state enums dispatched in `Update` (`sessionState`, `dialogState`)
- Complex constructors take a config struct (`DialogConfig`, `SingleSelectConfig`)

## Testing

Tests use stdlib `testing` only (no testify). White-box testing (same package).

### Patterns

- **Table-driven tests** with `t.Run` subtests
- **`t.Helper()`** in all helper functions
- **`t.TempDir()`** for filesystem tests (auto-cleanup)
- **`t.Errorf`** for non-fatal assertions, **`t.Fatalf`** when continuing would cause panics
- **Inline test data** — no `testdata/` directories; JSON embedded in test structs

```go
tests := []struct {
    name string
    json string
    want TextFilter
}{
    {name: "PlainString", json: `"hello"`, want: TextFilter{Value: "hello"}},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ...
        if got != tt.want {
            t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
        }
    })
}
```

## Important Notes

- **Go version**: 1.25+ (see `go.mod`)
- **Main branch**: `trunk`
- **CI**: GitHub Actions (`.github/workflows/auto.yml`) runs `bun install` in `web-ui`, `go generate ./internal/web/...`, `go mod download`, `go mod verify`, `go build -v ./...`, then `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
- **Commits**: [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`)
- **Debug logs**: `.lazylogcat.log` (gitignored)
- **Config**: Layered discovery — `~/.config/lazylogcat/config.json` → `.lazylogcat/config.json` → `.lazylogcat/config.local.json` (overridable with `--config` / `--config-local`)

## Key Dependencies

| Package | Purpose |
|---------|---------|
| charm.land/bubbletea/v2 v2.0.6 | TUI framework |
| charm.land/bubbles/v2 v2.1.0 | TUI components (viewport, textinput, table) |
| charm.land/lipgloss/v2 v2.0.3 | Terminal styling |
| github.com/spf13/cobra v1.10.2 | CLI framework |
| github.com/atotto/clipboard v0.1.4 | Cross-platform clipboard |
| github.com/pkg/browser | Auto-open browser for `lazylogcat web` |
| nhooyr.io/websocket v1.8.17 | WebSocket server in `internal/web` |
