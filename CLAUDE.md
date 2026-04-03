# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Monocular is a read-only TUI dashboard for observing [OpenCode](https://opencode.ai) server SSE event streams. It connects to `/global/event` on a running OpenCode server and displays real-time session activity, tool executions, permission requests, and event logs. It also supports `--json` mode for NDJSON output (no TUI).

## Build and test commands

```sh
make build          # build binary to ./monocular
make test           # go test ./...
make test-race      # go test -race ./...
make vet            # go vet ./...
make lint           # runs go vet
go test ./internal/model/...   # run tests for a single package
go test -run TestApplyEvent ./internal/model/...  # run a single test
```

## Architecture

This is a Bubble Tea v2 application following the Elm architecture (Model-Update-View).

### Data flow

SSE stream -> `sse.Connect()` -> channels -> Bubble Tea Cmd/Msg loop -> `model.ApplyEvent()` mutates state -> `ui.App.View()` renders

### Package responsibilities

- **`cmd/`** -- Cobra root command, CLI flag parsing. Decides between TUI mode and JSON mode.
- **`internal/sse/`** -- SSE client. `connection.go` manages the HTTP connection with exponential backoff reconnection. `parser.go` is a line-by-line SSE parser that assembles `GlobalEvent` structs. `events.go` defines event categories, color mapping, and badge labels. `reconnect.go` fetches session state via REST after reconnection.
- **`internal/model/`** -- Dashboard state and event processing. `state.go` defines `DashboardState` (the Elm Model). `update.go` contains `ApplyEvent()` which is a pure state mutation function that maps SSE events to state changes. `messages.go` defines Bubble Tea `Msg` types.
- **`internal/ui/`** -- TUI components. `app.go` is the root Bubble Tea model that composes all sub-components and handles keyboard input. Layout is three-column: sessions (left 30%), event log (center 40%), detail (right 30%). Sub-components: `connection_bar`, `session_panel`, `tool_tracker`, `event_log`, `detail_panel`, `alert_bar`, `help_bar`, `filter_picker`.
- **`internal/format/`** -- Display formatters for timestamps, costs, tokens, durations, and string truncation.
- **`internal/jsonmode/`** -- Non-TUI NDJSON output mode. Connects to SSE and writes filtered events as JSON lines to stdout.
- **`testdata/`** -- Shared test fixtures. `fixtures.go` provides builder functions for constructing sample `GlobalEvent` values and raw SSE wire-format strings.

### Key design decisions

- **State is centralized**: All dashboard state lives in `model.DashboardState`. UI components receive a pointer to it for rendering but do not own state.
- **`ApplyEvent` is pure-ish**: It takes `(state, event, now)` and mutates state, making it deterministic and testable without Bubble Tea.
- **SSE connection is decoupled from TUI**: Events and state changes flow through Go channels, bridged to Bubble Tea via `waitForSSEEvent`/`waitForSSEState` Cmd functions.
- **Event properties are `map[string]any`**: Parsed from JSON with helper extractors (`getString`, `getFloat`, `getMap`, `getSlice`) in `update.go`.

## Charm v2 specifics

This project uses **Bubble Tea v2**, **Lip Gloss v2**, and **Bubbles v2** from `charm.land/*` (not `github.com/charmbracelet/*`). Key differences from v1:

- `View()` returns `tea.View` (not `string`). Use `tea.NewView(str)` and set `v.AltScreen = true`.
- Bubbles components use getter/setter methods: `viewport.SetWidth(w)` not `viewport.Width = w`.
- Key messages are `tea.KeyPressMsg` (not `tea.KeyMsg`).
- Window size messages are `tea.WindowSizeMsg` (not `tea.WindowSizeMsg` -- same name, different package).

## Spec

`SPEC.md` contains the original implementation plan with the full event type catalog and UI layout specification. Consult it for the intended behavior of event handling and dashboard layout.
