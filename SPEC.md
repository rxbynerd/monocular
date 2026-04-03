# Implementation Plan: OpenCode SSE Stream Explorer TUI

> Drop this document into a Claude Code session to build the tool from scratch.
> Reference `API.md` in this repository for the broad API shape, but when it conflicts with `packages/sdk/openapi.json`, `packages/sdk/js/src/v2/gen/types.gen.ts`, or current route handlers under `packages/opencode/src/server/routes/`, prefer the current OpenAPI and route handlers. This tool consumes raw `/global/event` SSE, so current Bus event names are the source of truth.

---

## Goal

Build **`oc-explorer`**, a standalone Terminal User Interface that connects to a running OpenCode server's SSE event stream and presents a real-time visual dashboard of everything happening in the instance. This is a read-only diagnostic/observability tool -- it does not send prompts or modify state.

## Non-goals

- Sending prompts or commands to OpenCode (this is an observer, not a client)
- Replacing the built-in OpenCode TUI
- Supporting the sync-event stream (`/global/sync-event`) -- only `/global/event`
- Persisting event history to disk

---

## 1. Technology Stack

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Language | **Go 1.26+** | Single static binary, zero runtime deps, no supply-chain risk from npm/node_modules |
| TUI framework | **Bubble Tea v2** (`charm.land/bubbletea/v2`) | Elm-architecture TUI framework. v2.0.0 (Feb 2026) ships the new Cursed Renderer (ncurses-based), mode 2026 synchronized output, and progressive keyboard enhancements. |
| Styling | **Lip Gloss v2** (`charm.land/lipgloss/v2`) | Composable terminal styling (colors, borders, padding, alignment). v2 makes styles deterministic and pairs with Bubble Tea v2. |
| Common widgets | **Bubbles v2** (`charm.land/bubbles/v2`) | Pre-built Bubble Tea components: `viewport` (scrollable panes), `list` (selectable lists), `textinput` (search bar), `key` (key binding help). v2 uses getter/setter methods instead of exported Width/Height fields. |
| SSE client | **`net/http` + `bufio.Scanner`** | Go stdlib. No third-party SSE library needed -- the wire format is trivial to parse with a line scanner. Full control over reconnection and cancellation via `context.Context`. |
| JSON parsing | **`encoding/json`** | stdlib. Unmarshal into `map[string]any` for properties, typed structs for the envelope. |
| CLI flags | **`github.com/spf13/cobra`** v1.10.2 | Standard Go CLI framework. Provides subcommands, `--help` generation, flag parsing. Single well-audited dependency. Alternatively, use stdlib `flag` if you prefer zero deps. |
| Testing | **`testing` + `github.com/stretchr/testify`** v1.11.1 | stdlib test runner. testify for assertions/table-driven tests. |
| Build | **`go build`** | Single binary, cross-compile with `GOOS`/`GOARCH`. |

### Dependency summary (4 direct deps, all widely audited)

```
charm.land/bubbletea/v2              # TUI framework (v2.0.0, Feb 2026)
charm.land/lipgloss/v2               # Styling (v2.0.0, Feb 2026)
charm.land/bubbles/v2                # Widgets: viewport, list, textinput, key (v2.0.0, Feb 2026)
github.com/spf13/cobra               # CLI (v1.10.2, Dec 2025)
```

Dev/test only:
```
github.com/stretchr/testify          # Test assertions (v1.11.1)
charm.land/x/exp/teatest/v2          # Bubble Tea test harness (golden files, test model)
```

**Note on Charm v2 migration:** The entire Charm ecosystem moved module paths from `github.com/charmbracelet/*` to `charm.land/*` in the v2 release (Feb 23, 2026). All imports must use the new `charm.land` paths. Key v2 breaking changes: Bubbles components use getter/setter methods (e.g., `viewport.SetWidth()` instead of `viewport.Width = ...`); Lip Gloss v2 styles are deterministic; Bubble Tea v2 includes the Cursed Renderer by default with mode 2026 synchronized output.

---

## 2. Project Structure

```
tools/oc-explorer/
  go.mod
  go.sum
  main.go                        # Entry point: cobra root command, flag parsing
  cmd/
    root.go                      # Root command: parse flags, launch TUI or JSON mode
  internal/
    sse/
      connection.go              # SSE HTTP client with reconnect logic
      connection_test.go
      parser.go                  # Parse raw SSE lines into typed events
      parser_test.go
      events.go                  # Event type constants, category mapping, color mapping
      events_test.go
    model/
      state.go                   # Dashboard state types (Model in Elm terms)
      update.go                  # Event -> state transitions (Update in Elm terms)
      update_test.go
      messages.go                # Bubble Tea Msg types (SSE event received, connection state changed, tick)
    ui/
      app.go                     # Root Bubble Tea model: layout, delegates to sub-models
      app_test.go
      connection_bar.go          # Top bar: URL, connection state, uptime, event count, cost
      session_panel.go           # Left panel: session list with status badges
      session_panel_test.go
      tool_tracker.go            # Sub-panel below sessions: active tool executions
      event_log.go               # Center panel: scrollable event stream
      event_log_test.go
      detail_panel.go            # Right panel: selected event detail (formatted + raw JSON)
      alert_bar.go               # Alert row: permission/question/error notifications
      help_bar.go                # Bottom row: keybinding hints
      filter_picker.go           # Overlay: toggle event categories on/off
      styles.go                  # Lip Gloss style definitions (colors, borders, badges)
    format/
      format.go                  # Timestamp, cost, token, ID formatters
      format_test.go
      truncate.go                # String truncation for fixed-width columns
    jsonmode/
      jsonmode.go                # --json NDJSON output mode (no TUI)
      jsonmode_test.go
  testdata/
    fixtures.go                  # Sample event payloads for every event type
```

---

## 3. SSE Connection Layer

### 3.1 Wire format

The `/global/event` endpoint emits SSE in this format:
```
data: {"directory":"/path/to/project","payload":{"type":"session.status","properties":{"sessionID":"ses_01J...","status":{"type":"busy"}}}}\n\n
```

Bus-published events look like the example above. Route-generated `server.connected` and `server.heartbeat` events currently omit `directory`, so the parser must tolerate an empty/missing directory field.

Key facts from the current server and OpenAPI:
- **Envelope:** Usually `{ "directory": string, "payload": { "type": string, "properties": object } }`
- **Heartbeat:** `server.heartbeat` every 10 seconds (use to detect stale connections)
- **Initial event:** `server.connected` on first connect
- **No Last-Event-ID replay** -- server ignores it; we must re-fetch state on reconnect
- **No compression** on SSE endpoints
- **No server-side directory narrowing on `/global/event`** -- keep the subscription global and apply `--directory` as a client-side filter in the TUI / JSON output

### 3.2 Connection state machine

```
DISCONNECTED -> CONNECTING -> CONNECTED -> DISCONNECTED
                    |                          ^
                    v                          |
               CONNECT_FAILED ----[backoff]----+
```

States:
- `Disconnected` -- initial, or after explicit close
- `Connecting` -- HTTP request in flight
- `Connected` -- response body streaming, receiving events
- `ConnectFailed` -- request failed or stream broke; backoff before retry

### 3.3 `connection.go` specification

```go
type ConnectionState int

const (
    Disconnected ConnectionState = iota
    Connecting
    Connected
    ConnectFailed
)

// GlobalEvent is the envelope from /global/event.
type GlobalEvent struct {
    Directory string       `json:"directory"` // may be empty for route-generated infra events
    Payload   EventPayload `json:"payload"`
}

type EventPayload struct {
    Type       string         `json:"type"`
    Properties map[string]any `json:"properties"`
}

type ConnectionConfig struct {
    URL          string        // e.g. "http://127.0.0.1:4096/global/event"
    Directory    string        // Optional client-side directory filter; do not append to /global/event
    MaxRetries   int           // Default: 50
    StaleTimeout time.Duration // Default: 30s -- force reconnect if no event received
}

// Connect runs the SSE connection loop. It blocks until ctx is cancelled.
// Events are sent to eventCh. State changes are sent to stateCh.
// On stream break, it reconnects with exponential backoff.
func Connect(ctx context.Context, cfg ConnectionConfig, eventCh chan<- GlobalEvent, stateCh chan<- ConnectionState) error
```

Implementation notes:
- Use `http.NewRequestWithContext(ctx, "GET", url, nil)` for cancellation.
- Set headers: `Accept: text/event-stream`, `Cache-Control: no-cache`.
- Read response body with `bufio.NewScanner(resp.Body)`, split on lines.
- Buffer lines; on empty line (`""`), join buffered lines and parse as one SSE event.
- Exponential backoff on reconnect: 1s, 2s, 4s, 8s, capped at 30s. Use `time.After` with jitter.
- Detect stale connection: track `lastEventTime`. A background goroutine checks every 10s; if `time.Since(lastEventTime) > StaleTimeout`, close `resp.Body` to force reconnect.
- On reconnect, send a synthetic event with type `_reconnected` so the UI can show a gap marker.
- Do **not** append `?directory=...` to `/global/event`. Keep the stream global and apply `Directory` after parsing, in the UI / JSON mode.

### 3.4 `parser.go` specification

```go
// ParseSSELines processes raw SSE data. Call for each line from the scanner.
// Returns a complete GlobalEvent when a blank line terminates an event block.
// Returns nil, nil for partial/incomplete events (still buffering).
// Returns nil, err for malformed JSON.
type SSEParser struct {
    buf []byte // accumulated data: lines
}

func NewSSEParser() *SSEParser

// FeedLine processes one line. Returns (event, nil) when complete, (nil, nil) when buffering.
func (p *SSEParser) FeedLine(line string) (*GlobalEvent, error)
```

Logic:
- If line starts with `data: `, strip prefix and append to `buf` (with newline separator for multi-line data).
- If line is empty and `buf` is non-empty, `json.Unmarshal(buf)` into `GlobalEvent`, reset `buf`, return event.
- If line starts with `:` (SSE comment), ignore.
- If line starts with `id:` or `retry:`, ignore (server doesn't use these meaningfully).

### 3.5 `events.go` -- event type constants and categories

```go
type EventCategory string

const (
    CategorySession    EventCategory = "session"
    CategoryMessage    EventCategory = "message"
    CategoryPermission EventCategory = "permission"
    CategoryQuestion   EventCategory = "question"
    CategoryFile       EventCategory = "file"
    CategoryInfra      EventCategory = "infra"
    CategoryPTY        EventCategory = "pty"
    CategoryWorkspace  EventCategory = "workspace"
    CategoryTUI        EventCategory = "tui"
    CategoryTodo       EventCategory = "todo"
)

// Categorize returns the EventCategory for a given event type string.
func Categorize(eventType string) EventCategory

// CategoryColor returns the lipgloss.Color for a category.
func CategoryColor(cat EventCategory) lipgloss.TerminalColor

// CategoryBadge returns the short bracketed label, e.g. "[session]", "[perm]".
func CategoryBadge(cat EventCategory) string

// AllCategories returns all categories for the filter picker.
func AllCategories() []EventCategory
```

Known event types for the current raw `/global/event` stream:

```go
// Session lifecycle
"session.created", "session.updated", "session.deleted",
"session.status", "session.idle", "session.compacted",
"session.diff", "session.error"

// Messages
"message.updated", "message.removed",
"message.part.updated", "message.part.removed", "message.part.delta"

// Permissions
"permission.asked", "permission.replied"

// Questions
"question.asked", "question.replied", "question.rejected"

// Files & VCS
"file.edited", "file.watcher.updated", "vcs.branch.updated", "project.updated"

// Infrastructure
"server.connected", "server.heartbeat", "server.instance.disposed", "global.disposed",
"installation.updated", "installation.update-available",
"lsp.updated", "lsp.client.diagnostics", "mcp.tools.changed", "mcp.browser.open.failed",
"command.executed"

// PTY
"pty.created", "pty.updated", "pty.exited", "pty.deleted"

// Workspaces
"workspace.ready", "workspace.failed", "worktree.ready", "worktree.failed"

// TUI
"tui.prompt.append", "tui.command.execute", "tui.toast.show", "tui.session.select"

// Todos
"todo.updated"
```

Categorization: split on first `.` -- `"session.created"` -> `"session"` -> `CategorySession`. Special cases: `"server.*"`, `"installation.*"`, `"lsp.*"`, `"mcp.*"`, `"command.*"`, `"global.*"` all map to `CategoryInfra`. `"vcs.*"` and `"project.*"` map to `CategoryFile`.

---

## 4. Dashboard State (the Bubble Tea Model)

### 4.1 State shape (`model/state.go`)

```go
type DashboardState struct {
    // Connection
    Connection ConnectionInfo

    // Sessions (keyed by sessionID)
    Sessions     map[string]*SessionEntry
    SessionOrder []string // Ordered by last activity (most recent first)

    // Event log (ring buffer)
    Events    []EventLogEntry
    MaxEvents int // Default: 500

    // Alerts (permissions, questions, errors -- newest first)
    Alerts []Alert

    // Aggregate counters
    Counters Counters

    // UI focus/selection state
    UI UIState
}

type ConnectionInfo struct {
    State          ConnectionState
    URL            string
    ConnectedAt    time.Time
    ReconnectCount int
    LastEventAt    time.Time
}

type SessionEntry struct {
    ID            string
    Slug          string
    Title         string
    Directory     string
    Status        SessionStatus
    LastActivity  time.Time
    ActiveTools   []ToolExecution
    MessageCount  int
    TotalCost     float64
    TotalTokens   TokenCounts
}

type SessionStatus struct {
    Type    string // "idle", "busy", "retry"
    Attempt int    // only for retry
    Message string // only for retry
}

type ToolExecution struct {
    CallID    string
    Tool      string
    SessionID string
    Status    string // "pending" or "running"
    StartedAt time.Time
}

type TokenCounts struct {
    Input     int
    Output    int
    Reasoning int
}

type EventLogEntry struct {
    ID         int
    Timestamp  time.Time
    Directory  string // may be empty for route-generated infra events
    Type       string
    Category   EventCategory
    Summary    string // One-line human-readable
    Properties map[string]any
}

type Alert struct {
    ID         string
    Kind       string // "permission", "question", "error"
    SessionID  string
    Title      string // Derived one-line description shown in the alert bar
    Timestamp  time.Time
    Properties map[string]any
}

type Counters struct {
    TotalEvents  int
    EventsByType map[string]int
    TotalCost    float64
    TotalTokens  TokenCounts
    FilesEdited  map[string]struct{} // set
}

type UIState struct {
    SelectedSessionID string
    Directory         string // Optional exact directory filter applied client-side
    SelectedEventIdx  int
    Filter            map[EventCategory]bool // nil = show all
    DetailExpanded    bool
    FocusedPanel      Panel
    Paused            bool
    PauseBuffer       []EventLogEntry
    SearchQuery       string
    SearchActive      bool
    ShowHelp          bool
    ShowFilter        bool
}

type Panel int
const (
    PanelSessions Panel = iota
    PanelEvents
    PanelDetail
)
```

### 4.2 Update logic (`model/update.go`)

In Bubble Tea, `Update(msg) (Model, Cmd)` is the equivalent of a reducer. The `msg` types are defined in `messages.go`:

```go
// Custom Bubble Tea messages

type SSEEventMsg struct {
    Event GlobalEvent
}

type ConnectionStateMsg struct {
    State ConnectionState
}

type TickMsg struct{} // Fired every 1s for elapsed-time counters

// Built-in Bubble Tea v2 messages used in Update():
//   tea.KeyPressMsg      -- keyboard input (replaces v1's tea.KeyMsg)
//   tea.KeyReleaseMsg    -- key release (new in v2, requires Kitty protocol)
//   tea.WindowSizeMsg    -- terminal resize
```

Key state transitions:

| Event type | State mutation |
|------------|---------------|
| `server.connected` | Set connection state to `Connected`, record timestamp |
| `server.heartbeat` | Update `LastEventAt` only; do **not** add to event log |
| `session.created` | Add session to map with idle status; prepend to `SessionOrder` |
| `session.updated` | Update session info (title, summary) |
| `session.deleted` | Remove from map and `SessionOrder`. Current payload includes both `sessionID` and `info`; use `sessionID` when present, fall back to `info.id` if needed |
| `session.status` | Update session status. Status is always an object: `{type:"idle"}` / `{type:"busy"}` / `{type:"retry",...}` |
| `session.idle` | Set status to idle, clear `ActiveTools` for that session |
| `session.error` | Add alert with kind `"error"` |
| `message.updated` | Increment session `MessageCount`; if `role == "assistant"` and has `cost`/`tokens`, accumulate into session and global counters |
| `message.part.updated` | Extract `type` from part. If `type == "tool"`: check `state.status`; add to `ActiveTools` if `"pending"`/`"running"`, remove if `"completed"`/`"error"` |
| `message.part.delta` | Skip (do not add to event log -- too noisy). Update `LastEventAt` only. |
| `permission.asked` | Add alert with kind `"permission"`; derive the alert text from `permission` and `patterns` / metadata |
| `permission.replied` | Remove matching alert |
| `question.asked` | Add alert with kind `"question"` |
| `question.replied` / `question.rejected` | Remove matching alert |
| `file.edited` | Add path to `FilesEdited` set |
| `todo.updated` | Log only |
| `command.executed` | Log only |
| `*` (any non-skipped event) | Append to event log ring buffer (drop oldest if at capacity), increment counters |

**Summary generation** (for event log one-liners):
- `session.created` -> `"Session created: {title}"`
- `session.status` -> `"Session {slug}: {status.type}"`
- `session.deleted` -> `"Session deleted: {title}"`
- `message.updated` -> `"Message from {role} in {slug}"` (+ `" ($0.0042)"` if assistant with cost)
- `message.part.updated` -> `"[{part.type}] {tool name or first 60 chars of text}"`
- `permission.asked` -> `"Permission requested: {permission}"` (optionally append first pattern if present)
- `file.edited` -> `"File edited: {file}"`
- All others -> `"{type}"` (just the event type as fallback)

---

## 5. UI Layout

```
+--[Connection: http://127.0.0.1:4096 | CONNECTED | 5m23s | 847 events | $0.42]--+
|                          |                               |                       |
|  Sessions (4)            |  Event Stream                 |  Detail               |
|                          |                               |                       |
|  > [BUSY] Fix auth bug   |  12:34:05 [session] status:   |  session.status        |
|    [IDLE] Refactor DB     |           busy ses_01J..      |  ---                   |
|    [IDLE] Add tests       |  12:34:06 [message] part:     |  sessionID: ses_01J... |
|    [RETRY] Deploy fix     |           [tool] bash running |  status:               |
|                          |  12:34:07 [file] edited:      |    type: busy          |
|  -- Active Tools --      |           src/auth/index.ts   |                       |
|  bash (ses_01J..) 3.2s   |  12:34:08 [perm] requested:  |  Raw JSON:             |
|  read (ses_01J..) 0.1s   |           bash: npm install   |  { "type": "session... |
|                          |  12:34:09 [message] part:     |                       |
|                          |           [text] "I'll fix..." |                       |
+----------+---------------+-------------------------------+-----------------------+
| [!] Permission pending: bash "npm install" in ses_01J... (3s ago)                |
+----------+--------------------------------------------------------------------+-+
| q:quit  tab:panel  j/k:scroll  enter:expand  f:filter  c:clear  ?:help          |
+----------------------------------------------------------------------------------+
```

### Layout breakdown

In Bubble Tea, layout is built by composing strings in the `View()` method. Use `lipgloss.JoinHorizontal` for columns and `lipgloss.JoinVertical` for stacking rows.

| Region | Component | Size |
|--------|-----------|------|
| Top bar | `ConnectionBar.View()` | 1 row, full width |
| Left panel | `SessionPanel.View()` + `ToolTracker.View()` | 30% of terminal width |
| Center panel | `EventLog.View()` | 40% of terminal width |
| Right panel | `DetailPanel.View()` | 30% of terminal width |
| Alert bar | `AlertBar.View()` | 1-2 rows, full width (hidden when no alerts) |
| Bottom bar | `HelpBar.View()` | 1 row, full width |

Each panel is a sub-model with its own `Update()` and `View()`. The root `App` model delegates key events to the focused panel.

Scrollable panels (`EventLog`, `DetailPanel`, `SessionPanel`) use `bubbles/viewport` internally for scrolling and line management.

### Color scheme (Lip Gloss styles in `ui/styles.go`)

| Category | Color | Badge |
|----------|-------|-------|
| session | `lipgloss.Color("6")` (cyan) | `[session]` |
| message | `lipgloss.Color("2")` (green) | `[message]` |
| permission | `lipgloss.Color("3")` (yellow, bold) | `[perm]` |
| question | `lipgloss.Color("5")` (magenta) | `[question]` |
| file | `lipgloss.Color("4")` (blue) | `[file]` |
| infra | `lipgloss.Color("8")` (dim gray) | `[infra]` |
| pty | `lipgloss.Color("7")` (white) | `[pty]` |
| error/alert | `lipgloss.Color("1")` (red, bold) | `[error]` |
| tool pending | dim | |
| tool running | yellow | |
| tool completed | green | |
| tool error | red | |
| session idle | green | `[IDLE]` |
| session busy | yellow | `[BUSY]` |
| session retry | red | `[RETRY]` |
| focused panel border | bright/highlighted | |
| unfocused panel border | dim | |

---

## 6. Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus between panels (sessions -> events -> detail) |
| `j` / `Down` | Scroll down in focused panel |
| `k` / `Up` | Scroll up in focused panel |
| `g` / `Home` | Jump to top of focused panel |
| `G` / `End` | Jump to bottom of focused panel |
| `Enter` | In event log: show full detail in right panel. In sessions: select and filter events to that session. |
| `Escape` | Clear selection / collapse detail / close overlay |
| `f` | Toggle filter picker overlay |
| `c` | Clear event log |
| `p` | Pause/resume event ingestion (buffer events while paused) |
| `/` | Activate search input (filter event log by text) |
| `1`-`3` | Jump to panel by number |
| `q` / `Ctrl+C` | Quit |
| `?` | Toggle help overlay |

In Bubble Tea v2, key messages are split into `tea.KeyPressMsg` and `tea.KeyReleaseMsg`. Use `tea.KeyPressMsg` for input handling and `msg.String()` for matching:

```go
case tea.KeyPressMsg:
    switch msg.String() {
    case "tab":
        m.cycleFocus(1)
    case "shift+tab":
        m.cycleFocus(-1)
    case "q", "ctrl+c":
        return m, tea.Quit
    case "j", "down":
        m.focusedPanel().ScrollDown(1)
    // ...
    }
```

Note: In v2, `key.Type`/`key.Runes` are replaced by `key.Code`/`key.Text`, and modifiers live in `key.Mod`. The `msg.String()` switch pattern is still the recommended approach for simple key matching. For more complex bindings, use `key.Matches(msg, binding)` from `charm.land/bubbles/v2/key`.

---

## 7. CLI Interface

```
oc-explorer [flags]

Flags:
  -u, --url string        OpenCode server URL (default "http://127.0.0.1:4096")
  -d, --directory string  Initial client-side directory filter (does not change the SSE subscription scope)
  -f, --filter string     Comma-separated event categories to show (default: all)
                          Categories: session,message,permission,question,file,infra,pty,workspace,tui,todo
      --no-color          Disable colors
      --json              Output raw events as NDJSON to stdout (no TUI, for piping)
  -h, --help              Show help
  -v, --version           Show version

Examples:
  oc-explorer                                        # Connect to default local server
  oc-explorer --url http://192.168.1.5:4096          # Connect to remote server
  oc-explorer -d /Users/me/myproject                 # Start with one project selected in the UI
  oc-explorer -f session,message,permission          # Only show these categories
  oc-explorer --json | jq '.payload.type'            # Pipe raw events
```

`--json` mode skips the TUI entirely and writes each `GlobalEvent` as a JSON line to stdout. This enables piping into `jq`, `grep`, log files, or other tools. If `--directory` is set, apply it client-side before writing each line.

Implementation in `cmd/root.go` using cobra:

```go
var rootCmd = &cobra.Command{
    Use:   "oc-explorer",
    Short: "Real-time TUI dashboard for OpenCode SSE events",
    RunE: func(cmd *cobra.Command, args []string) error {
        if jsonMode {
            return jsonmode.Run(ctx, cfg)
        }
        return runTUI(cfg)
    },
}
```

---

## 8. Implementation Steps

Execute these in order. Each step should result in a working (if incomplete) state. Commit after each step.

### Step 1: Project scaffolding
- Create `tools/oc-explorer/` with `go.mod` (`module github.com/<org>/opencode/tools/oc-explorer`).
- `go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 charm.land/bubbles/v2 github.com/spf13/cobra@v1.10.2`.
- Create `main.go` with cobra root command that parses `--url`, `--directory`, `--filter`, `--json`, `--no-color`.
- Verify `go run . --help` prints usage.
- Verify `go build -o oc-explorer .` produces a binary.

### Step 2: SSE parser and connection
- Implement `internal/sse/parser.go` with `SSEParser` and `FeedLine()`.
- Implement `internal/sse/events.go` with event type constants and `Categorize()`.
- Implement `internal/sse/connection.go` with `Connect()`.
- Write `internal/sse/parser_test.go` and `internal/sse/connection_test.go`.
- Implement `internal/jsonmode/jsonmode.go` for `--json` mode.
- Verify by running `go run . --json` against a live OpenCode server -- events should print as NDJSON.

### Step 3: State management
- Implement `internal/model/state.go` (dashboard state types).
- Implement `internal/model/messages.go` (Bubble Tea Msg types).
- Implement `internal/model/update.go` (event -> state transitions).
- Write `internal/model/update_test.go` with fixture events covering every event type.
- Implement `internal/format/format.go` and `internal/format/truncate.go`.
- Write `internal/format/format_test.go`.

### Step 4: Minimal TUI shell
- Implement `internal/ui/styles.go` (all Lip Gloss styles).
- Implement `internal/ui/app.go` -- root Bubble Tea model with `Init()`, `Update()`, `View()`.
  - `Init()`: start SSE connection goroutine, return initial tick command.
  - `Update()`: dispatch `SSEEventMsg` to state update, `tea.KeyPressMsg` to key handler, `tea.WindowSizeMsg` to resize handler.
  - `View()`: compose top bar + three columns + alert bar + bottom bar.
- Implement `internal/ui/connection_bar.go` (renders connection state, uptime, event count, cost).
- Implement `internal/ui/help_bar.go` (renders keybinding hints).
- Wire it all together: `cmd/root.go` starts `tea.NewProgram(ui.NewApp(cfg))`.
- Verify: app renders, connects to server, top bar shows `CONNECTED`, bottom bar shows keys.

### Step 5: Event log panel
- Implement `internal/ui/event_log.go` using `bubbles/viewport` for scrolling.
- `View()` renders filtered event entries with category badges, timestamps, summaries.
- Handle `j`/`k`/`Up`/`Down` for scrolling, `Enter` for selecting.
- New events auto-scroll to bottom unless user has scrolled up (follow mode).
- Verify: events stream into the log in real time with colored badges.

### Step 6: Session panel
- Implement `internal/ui/session_panel.go` -- list of sessions with `[IDLE]`/`[BUSY]`/`[RETRY]` badges.
- Implement `internal/ui/tool_tracker.go` -- renders below sessions, shows `tool (session) elapsed`.
- `Enter` on a session filters the event log to that session's events.
- Verify: sessions appear/disappear, status badges update, tool list is live.

### Step 7: Detail panel and alerts
- Implement `internal/ui/detail_panel.go` -- shows selected event's formatted properties and raw JSON.
- Implement `internal/ui/alert_bar.go` -- renders active permission/question/error alerts with elapsed time.
- Verify: selecting an event shows its full payload; `permission.asked` events trigger alerts.

### Step 8: Filtering, search, pause
- Implement `internal/ui/filter_picker.go` -- overlay listing categories with checkboxes, toggled with `Space`.
- Wire `f` to toggle filter overlay, `p` for pause, `/` to activate `bubbles/textinput` search.
- `c` clears the event log.
- Wire `--filter` and `--directory` CLI flags to initial state. `--directory` is a client-side visibility filter for sessions, event log, tool tracker, alerts, and JSON mode output.
- Verify all interactive controls work.

### Step 9: Polish and edge cases
- Handle `tea.WindowSizeMsg` -- recalculate column widths on terminal resize.
- Handle extremely rapid events: the SSE goroutine sends on a channel; if the channel is full, drop oldest. This prevents the TUI from being overwhelmed.
- Handle very long event properties: truncation in event log, word-wrap in detail panel.
- `TickMsg` every 1s updates elapsed time on active tools and alerts without needing new events.
- Test with a session that generates hundreds of rapid tool calls.
- Test reconnection by killing and restarting the server.

### Step 10: Final build and cross-compile
- Add `Makefile` or build script for `GOOS=darwin GOARCH=arm64 go build -o oc-explorer-darwin-arm64`.
- Test on macOS (arm64), Linux (amd64) at minimum.
- Run `go vet ./...` and `staticcheck ./...` (if available).

---

## 9. Test Plan

### Unit tests (run with `go test ./...`)

| Test file | What it covers | Cases |
|-----------|---------------|-------|
| `parser_test.go` | SSE line parsing | Valid `data:` lines; partial chunks across multiple `FeedLine` calls; empty lines (event boundaries); malformed JSON (returns error); multi-line data fields; comment lines (`:` prefix, ignored); `id:` and `retry:` fields (ignored); heartbeat event parsing |
| `events_test.go` | Event categorization | Every known event type maps to the correct category; unknown types get a sensible default (`CategoryInfra`); badge and color functions return non-empty values for all categories |
| `update_test.go` | State transitions | Table-driven tests with one case per event type from the catalog. Key scenarios: full session lifecycle (`session.created` -> `session.status {busy}` -> `session.idle` -> `session.deleted`); tool state machine (`message.part.updated` with `pending` -> `running` -> `completed` tool states); alert lifecycle (`permission.asked` -> `permission.replied`); ring buffer overflow (501st event drops oldest); cost/token accumulation from `message.updated` with assistant role; `session.deleted` prefers `sessionID` and falls back to `info.id`; `session.status` with all three object variants; `message.part.delta` is skipped from event log; `server.heartbeat` updates `LastEventAt` without log entry |
| `format_test.go` | Display formatters | Relative timestamps (`"3s ago"`, `"2m15s ago"`, `"1h ago"`); cost formatting (`"$0.0042"` vs `"$1.23"`); token count formatting (`"1,234"` vs `"1.2M"`); string truncation at various widths; session ID shortening (`"ses_01JXYZ..."` -> `"ses_01JX.."`) |
| `connection_test.go` | SSE connection | Uses `net/http/httptest.Server` to mock SSE. Tests: successful connect and event receipt; reconnect on stream close (verify backoff timing); context cancellation stops the loop; stale connection detection (no events for 30s triggers reconnect); non-200 response triggers `ConnectFailed` state; route-generated events with no `directory` field decode cleanly |
| `jsonmode_test.go` | NDJSON output | Captures stdout, feeds events, verifies one JSON object per line; verifies clean handling of `SIGPIPE` (broken pipe) |

### TUI integration tests

| Test file | What it covers |
|-----------|---------------|
| `app_test.go` | Uses `teatest.NewTestModel()` from `charm.land/x/exp/teatest/v2`. Tests: initial render contains connection bar and help bar; sending `SSEEventMsg` adds entry to event log; `Tab` cycles focus; `q` produces `tea.Quit` command; `WindowSizeMsg` doesn't panic. Use `teatest.RequireEqualOutput()` for golden-file snapshot tests. |
| `session_panel_test.go` | Sends session.created/status/deleted events, verifies rendered output contains correct badges and titles |
| `event_log_test.go` | Sends events, verifies scrolling works, filter hides events, search highlights matches |

### Manual integration tests

| Scenario | Procedure |
|----------|-----------|
| **Live connection** | Start `opencode serve`, run `oc-explorer`, trigger some AI prompts in another terminal, verify events appear in real time. |
| **Reconnection** | Start server, connect explorer, kill server (`Ctrl+C`), verify explorer shows `CONNECT_FAILED` then `CONNECTING`, restart server, verify `CONNECTED` resumes and gap marker appears. |
| **High throughput** | Trigger a session that scans a large repo or runs many tool calls in quick succession; verify the TUI remains responsive and the drop policy behaves predictably under load. |
| **JSON mode pipe** | `./oc-explorer --json --url ... \| head -5` -- verify 5 clean JSON lines and clean exit (no panic on broken pipe). |
| **Cross-platform** | Build for linux/amd64, run in a Docker container against a host-network OpenCode server. |

### Test fixtures (`testdata/fixtures.go`)

Export a `SampleEvents` map with one example `GlobalEvent` per event type. These are used across all unit tests:

```go
var SampleEvents = map[string]GlobalEvent{
    "session.created": {
        Directory: "/Users/dev/myproject",
        Payload: EventPayload{
            Type: "session.created",
            Properties: map[string]any{
                "sessionID": "ses_01JTEST",
                "info": map[string]any{
                    "id": "ses_01JTEST", "slug": "fix-auth-bug", "title": "Fix auth bug",
                    "projectID": "prj_01J", "directory": "/Users/dev/myproject",
                    "version": "1.0.0",
                    "time": map[string]any{"created": float64(time.Now().UnixMilli()), "updated": float64(time.Now().UnixMilli())},
                },
            },
        },
    },
    // ... one per event type
}
```

---

## 10. API Integration Gotchas

These are distilled from `API.md` pitfalls and are directly relevant to this tool's implementation. Keep these in mind during development.

1. **Heartbeat is the liveness signal.** `server.heartbeat` fires every 10s. If 30s pass with no event of any kind, the connection is dead -- force reconnect. Do not rely on TCP keepalive.

2. **`server.connected` is your handshake.** It fires once on initial connection. Use it to confirm the connection is live and to trigger an initial state fetch if needed. On reconnect (after stream break), you get a new `server.connected`.

3. **`message.part.delta` is high-frequency noise.** These fire for every text chunk during streaming. Filter them out of the event log. They contain `{ "field", "delta" }` where `delta` is a string fragment. Only update `LastEventAt`.

4. **`session.status` uses object variants, not strings.** All three states are `{"type":"idle"}`, `{"type":"busy"}`, `{"type":"retry",...}`. When unmarshalling into `map[string]any`, the `status` value is a `map[string]any` with a `"type"` key. Never compare the status value directly as a string.

5. **`session.deleted` currently carries both `sessionID` and `info`.** Use `sessionID` when present. If stale docs omit it, fall back to `properties["info"].(map[string]any)["id"]`.

6. **`session.diff` field is `diff` (singular).** Not `diffs`. The payload is `{"sessionID": ..., "diff": [...]}`.

7. **Do not conflate `permission.asked` with older `permission.updated` docs / clients.** Current raw `/global/event` SSE in this repo emits `permission.asked`, and its payload is a `PermissionRequest` (`id`, `sessionID`, `permission`, `patterns`, `metadata`, `always`, optional `tool`). Older docs and older generated clients may mention `permission.updated` with a flatter payload. This tool consumes raw SSE directly, so `permission.asked` is the source of truth.

8. **Directory filtering is client-side for `/global/event`.** The endpoint is global. Keep the subscription global and filter by `event.directory` in the TUI / JSON output. Do not treat `--directory` as an API query parameter for `/global/event`.

9. **The directory field may be empty on route-generated infra events.** `server.connected` and `server.heartbeat` are currently emitted directly from the route handler and may omit `directory`. Treat an empty string as valid for these events.

10. **`file.edited` uses `file`, not `path`.** The payload is `{"file": string}`. Use that field name in summaries, counters, and tests.

11. **No replay on reconnect.** After reconnection, any events during the gap are lost. For sessions panel accuracy, consider fetching `GET /session/status` on reconnect to refresh all session states. This is a REST call, not part of the SSE stream.

12. **`AssistantMessage` carries cost and token data.** When `message.updated` fires with `role == "assistant"`, extract `cost` (float64, USD) and `tokens` (`{"input": N, "output": N, "reasoning": N, "cache": {"read": N, "write": N}}`) for the dashboard counters. These are in `properties["info"].(map[string]any)`.

---

## 11. Release Guidance

### Build

```bash
# Development
cd tools/oc-explorer
go run . --url http://127.0.0.1:4096

# Production binary
go build -ldflags="-s -w" -o oc-explorer .

# Cross-compile
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o oc-explorer-darwin-arm64 .
GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o oc-explorer-darwin-amd64 .
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o oc-explorer-linux-amd64 .
GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o oc-explorer-linux-arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o oc-explorer-windows-amd64.exe .
```

Expected binary size: ~8-12MB (static, no CGO).

### Pre-release checklist

- [ ] All unit tests pass (`go test ./...`)
- [ ] `go vet ./...` clean
- [ ] No race conditions (`go test -race ./...`)
- [ ] Manual test: connect to live OpenCode server, observe session lifecycle events
- [ ] Manual test: reconnection after server restart
- [ ] Manual test: `--json` mode pipes cleanly to `jq`
- [ ] Manual test: `--directory` filter works
- [ ] Manual test: `--filter` category filter works
- [ ] Manual test: `--no-color` works
- [ ] Terminal resize doesn't crash
- [ ] Clean exit on `q` and `Ctrl+C` (no dangling goroutines, connection closed)
- [ ] Works in terminals: iTerm2, Terminal.app, VS Code integrated terminal, tmux
- [ ] Binary runs on macOS arm64, Linux amd64 (minimum targets)

### Versioning

- `0.1.0` -- MVP with all panels, keyboard nav, filtering
- `0.2.0` -- Reconnection state recovery (fetch `/session/status` on reconnect via REST)
- `0.3.0` -- Event recording/playback mode (save/load NDJSON files)

### Distribution

For now, build and run from `tools/oc-explorer/`. Copy the binary wherever needed -- it has zero runtime dependencies. If this proves useful, consider:
- Adding to the OpenCode install bundle
- Publishing binaries via GitHub Releases
- Adding a Homebrew formula

---

## 12. Open Questions

Decisions to make during implementation, not before:

1. **Should reconnection trigger a REST call to `/session/status`?** This would give accurate session states after a gap, but adds an HTTP client dependency to what's otherwise a pure SSE consumer. Start without it; add in v0.2.0 if stale-state is noticeable.

2. **Should `message.part.delta` events be shown at all?** They're noisy but show real-time text streaming. Option: show them only when a specific session is selected, collapsed into a single `"streaming..."` indicator otherwise.

3. **Should the tool tracker show elapsed time with a live-updating counter?** The `TickMsg` fires every 1s to update elapsed times. This is cheap in Go (no full re-render like React -- Bubble Tea only re-renders on Msg). Recommend: yes, enable it from the start.

4. **Should we support connecting to multiple OpenCode servers?** The `/global/event` stream already multiplexes by directory, so one server connection covers all instances on that server. Multiple server support would require multiple goroutines and a server-selector UI. Defer unless needed.

5. **`cobra` vs stdlib `flag`?** Cobra adds one dependency but gives better `--help` output, flag grouping, and shell completion for free. If zero-dep purity matters more, use stdlib `flag` -- the CLI surface is small enough.
