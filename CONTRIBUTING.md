# Contributing

Thanks for contributing to `mcp-proxy`.

## Development Environment

- Go `1.26`
- Make targets are the primary workflow wrapper

## Common Commands

```sh
make bootstrap
make fmt
make test
make lint
make build
```

## Project Structure

- `cmd/mcp-proxy`: binary entrypoint
- `internal/config`: config parsing/validation
- `internal/proxy`: process manager and stdio RPC bridge
- `internal/httpserver`: HTTP control-plane and per-process listeners
- `internal/tui`: Bubble Tea/Bubbles debug console

## Coding Standards

- Keep changes small and focused.
- Prefer clear names over clever implementations.
- Add GoDoc comments for exported symbols.
- Keep comments practical and behavior-focused.
- Preserve cross-platform behavior (Windows + non-Windows paths/commands).

## Testing

- Add or update tests for behavior changes.
- Run `make test` locally before opening a PR.
- At minimum, ensure `go test ./...` passes.

## Config and Runtime Changes

If you change runtime behavior, update:

- `README.md` (user-facing behavior)
- example config files (`mcp-proxy.example.json`)
- API docs in code comments where relevant

## Pull Request Guidance

- Explain what changed and why.
- Open an issue before opening a PR.
- Reference the issue number in the PR title or description using `(#Number)`.
- Note any backward-incompatible behavior.
- Include manual verification steps (commands and expected results).
- Call out any known follow-up work.
