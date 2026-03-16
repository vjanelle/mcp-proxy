# mcp-proxy

`mcp-proxy` is a local MCP stdio process manager with HTTP frontends and a
debug TUI.

It is designed for local MCP development workflows where you need to:
- run one or more stdio MCP servers under supervision
- expose them over stable local HTTP endpoints
- stop/restart processes quickly while iterating
- inspect request/response traffic and lifecycle events in real time

## Features

- Process lifecycle control (`start`, `stop`, `restart`) per MCP.
- Stdio transport bridging for JSON-RPC over:
  - `content-length` framed payloads
  - `newline` framed payloads
- Main control-plane HTTP API (`/v1/...`) plus per-process optional ports.
- Debug TUI built with Bubble Tea/Bubbles/Lip Gloss.
- Event ring buffer with API and TUI visibility.

## Requirements

- Go `1.26`

## Install

From this repository:

```sh
go install ./cmd/mcp-proxy
```

From module path:

```sh
go install github.com/vjanelle/mcp-proxy/cmd/mcp-proxy@latest
```

## Quick Start

1. Create `mcp-proxy.json`:

```json
{
  "listenAddr": "127.0.0.1:18080",
  "allowRemote": false,
  "apiKey": "",
  "apiKeyEnv": "",
  "maxRPCBodyBytes": 1048576,
  "processes": [
    {
      "name": "mcp",
      "command": "mcp.exe",
      "args": [],
      "workDir": "",
      "env": [],
      "autoStart": true,
      "port": 18081,
      "transport": "newline"
    }
  ]
}
```

2. Run:

```sh
make run
```

3. Send a proxied MCP request:

```sh
curl -X POST http://127.0.0.1:18081/rpc \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

## Architecture

- `cmd/mcp-proxy`: program entrypoint and server bootstrap.
- `internal/config`: JSON config loading and normalization.
- `internal/proxy`: process manager, RPC routing, event/status snapshots.
- `internal/httpserver`: control-plane and per-process HTTP servers.
- `internal/tui`: interactive console for process and log inspection.

## Configuration

Top-level fields:
- `listenAddr`: main API listener, default `127.0.0.1:18080`.
- `allowRemote`: set `true` to allow non-loopback listener addresses.
- `apiKey`: optional static API key for non-health endpoints (`X-API-Key` header).
- `apiKeyEnv`: optional env var name that provides `apiKey`.
- `maxRPCBodyBytes`: maximum request body size for RPC endpoints (default `1048576`).
- `processes`: array of process definitions.

Process fields:
- `name`: unique process identifier.
- `command`: executable path or name.
- `args`: command arguments.
- `workDir`: optional working directory.
- `env`: optional `KEY=VALUE` environment entries.
- `autoStart`: whether to launch on proxy startup.
- `port`: optional dedicated HTTP port for this process.
- `transport`: stdio framing mode:
  - `content-length` (default)
  - `newline`

## HTTP API

Main control-plane server (`listenAddr`):
- `GET /health`
- `GET /v1/processes`
- `GET /v1/events?process={name}`
- `POST /v1/processes/{name}/start`
- `POST /v1/processes/{name}/stop`
- `POST /v1/processes/{name}/restart`
- `POST /v1/processes/{name}/rpc`
- `POST /mcp/{name}` (alias for process RPC)

Per-process dedicated server (`port` set):
- `GET /health`
- `POST /rpc`
- `POST /` (alias for `/rpc`)

Authentication:
- When `apiKey` is set, all non-health endpoints require header `X-API-Key: <apiKey>`.
- `/health` stays unauthenticated.
- Non-loopback `listenAddr` requires `allowRemote: true` and a non-empty API key (via `apiKey` or `apiKeyEnv`).

## TUI Controls

- `tab`: switch focus between process table and debug log.
- `up/down` or `k/j`: navigate focused pane.
- `g`/`home`: jump to top of log (log pane).
- `G`/`end`: jump to bottom and re-enable follow-tail (log pane).
- `[` / `]`: resize left-right split.
- `-` / `=`: resize top-bottom split.
- `s`: start selected process.
- `x`: stop selected process.
- `r`: restart selected process.
- `q`: quit.

Mouse support:
- click to focus/select pane content
- mouse wheel to move through log selection

## Development

```sh
make bootstrap
make fmt
make test
make lint
make build
```

## Notes

- The TUI is optional. Run `mcp-proxy -no-tui` to keep only HTTP services.
- Some MCP servers require `transport: "newline"` instead of
  `content-length`. If requests time out with no response events, verify the
  transport mode.
- TUI log rendering escapes terminal control sequences for safer local debugging.
  Stored events and `/v1/events` payloads remain raw.
