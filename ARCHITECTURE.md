# Architecture

## Overview

`mcp-proxy` supervises local stdio MCP servers and exposes them over HTTP while
providing an interactive debug TUI.

Core runtime flow:

1. Load JSON config.
2. Construct `proxy.Manager` for configured processes.
3. Start auto-start processes.
4. Start HTTP servers:
   - main control-plane (`listenAddr`)
   - optional per-process listeners (`port`)
5. Start TUI (unless `-no-tui`).

## Components

### `internal/config`

Parses and validates runtime configuration.

Responsibilities:
- normalize defaults (`listenAddr`, process `transport`)
- validate required process fields (`name`, `command`)

### `internal/proxy`

Process supervision and JSON-RPC routing layer.

Responsibilities:
- spawn/stop/restart child MCP processes
- stdio framing (`content-length` or `newline`)
- correlate request IDs to responses
- expose process status snapshots and event history

### `internal/httpserver`

HTTP frontends for manager operations.

Main server:
- health/status/events
- lifecycle operations
- RPC forwarding by process name

Per-process server:
- dedicated `POST /rpc` endpoint for one process

### `internal/tui`

Bubble Tea/Bubbles operator console.

Responsibilities:
- process table
- debug log viewport + selected-line inspector
- keyboard/mouse control for focus, navigation, lifecycle actions
- resizable split panes

## Data Flow

RPC path:

1. HTTP request reaches control or per-process endpoint.
2. Handler forwards payload to `Manager.DoRPC`.
3. Managed process writes request to stdio transport.
4. Response is parsed from stdio and matched by JSON-RPC `id`.
5. HTTP handler returns response payload to caller.

Observability path:

1. Runtime emits lifecycle/request/response/error events.
2. Manager stores a bounded in-memory event ring.
3. Events are surfaced through:
   - `GET /v1/events`
   - TUI log viewport

## Concurrency Model

- One `managedProcess` per configured MCP.
- Each process uses dedicated goroutines for:
  - stdout read loop
  - stderr read loop
  - process wait/exit monitoring
- Manager/state synchronization uses mutexes and atomic counters.

## Operational Notes

- Some MCP servers require `transport: "newline"` instead of
  `content-length`.
- Per-process `port` endpoints simplify client configuration and debugging.
- TUI is optional for headless operation via `-no-tui`.

