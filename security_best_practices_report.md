# Security Best Practices Report

## Executive Summary

This Go codebase is clean and intentionally local-first, but it currently has several server-hardening gaps that can become high impact when the service is exposed beyond loopback (accidentally or intentionally).

Top risks are denial of service from unbounded request bodies and incomplete `http.Server` timeout/size controls. A secondary risk is that the control plane is unauthenticated and can execute process lifecycle/RPC actions if bound to a non-local interface. Event logging also captures full request/response payloads and process output, which can leak secrets.

## Critical Findings

No critical findings were identified in this pass.

## High Severity Findings

### SBP-001
- Rule ID: GO-HTTP-001
- Severity: High
- Location: `internal/httpserver/server.go:33-37`, `internal/httpserver/process_server.go:31-35`
- Evidence:
  - `http.Server` only sets `ReadHeaderTimeout`.
  - `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` are not configured.
- Impact: Slowloris and resource-exhaustion style clients can hold connections or abuse oversized headers/body behavior more easily, reducing service availability.
- Fix: Set explicit production-safe values for `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` in both server constructors.
- Mitigation: Keep listeners strictly on loopback and place a hardened reverse proxy with request/connection limits in front.
- False positive notes: If this service is guaranteed local-only and never remotely reachable, external exploitability is reduced but local abuse risk remains.

### SBP-002
- Rule ID: GO-HTTP-002
- Severity: High
- Location: `internal/httpserver/server.go:146-151`, `internal/httpserver/process_server.go:68-73`
- Evidence:
  - Request bodies are read with `io.ReadAll(request.Body)` without `http.MaxBytesReader` or any explicit cap.
- Impact: An attacker can send very large request bodies to exhaust memory and crash or degrade the proxy.
- Fix: Wrap `request.Body` with `http.MaxBytesReader` and enforce a conservative JSON-RPC payload limit (for example 1-4 MiB, configurable).
- Mitigation: Add ingress/body-size limits at the reverse proxy/firewall layer.
- False positive notes: Even on loopback this remains exploitable by any local process/user.

### SBP-003
- Rule ID: GO-AUTH-LOCAL-001 (project-specific)
- Severity: High
- Location: `internal/httpserver/server.go:83-170`, `internal/httpserver/process_server.go:62-92`, `cmd/mcp-proxy/main.go:98-113`
- Evidence:
  - Control and RPC endpoints (`/v1/processes/*`, `/mcp/{name}`, `/rpc`) have no authentication/authorization checks.
  - Process servers inherit host from `listenAddr` and can bind broadly if configured (`main.go` host derivation).
- Impact: If the service is exposed on a non-loopback interface, network clients can start/stop/restart managed processes and issue arbitrary proxied RPC calls.
- Fix: Enforce one or more of: loopback-only validation at startup, shared secret/API key auth middleware, or mTLS/reverse-proxy auth in front of all non-health endpoints.
- Mitigation: Keep `listenAddr` on `127.0.0.1` and avoid configuring public binds.
- False positive notes: README positions this as local development (`README.md:3-10`, `README.md:73`), so this may be accepted risk if exposure is strictly prevented operationally.

## Medium Severity Findings

### SBP-004
- Rule ID: GO-CONFIG-001
- Severity: Medium
- Location: `internal/proxy/process.go:170`, `internal/proxy/process.go:224`, `internal/proxy/process.go:263`, `internal/httpserver/server.go:72-80`
- Evidence:
  - Full RPC payloads and process stdout/stderr are emitted into events (`eventRequest`, `eventStdOut`, `eventResponse`).
  - Events are retrievable via `GET /v1/events` without redaction logic.
- Impact: Sensitive values in RPC parameters/results or process logs (tokens, credentials, internal data) can be exposed through the events API and TUI logs.
- Fix: Add structured redaction for known sensitive keys/patterns and optionally disable payload logging by default.
- Mitigation: Restrict event endpoint access to trusted local users/processes.
- False positive notes: If upstream MCP traffic never contains secrets, practical risk is lower.

## Low Severity Findings

No low-severity findings were identified in this pass.

## Positive Observations

- Default listener is loopback (`internal/config/config.go:9`), which is a strong secure default for local tooling.
- Command execution uses `exec.Command` arg arrays (not shell `-c`), reducing classic command injection risk from configuration (`internal/proxy/process.go:66`).

## Suggested Next Steps

1. Add body-size limits and full server timeout/header settings in both HTTP server constructors.
2. Decide and enforce an explicit trust model for remote exposure (block non-loopback by default or require auth).
3. Add log/event redaction or payload logging opt-out.
