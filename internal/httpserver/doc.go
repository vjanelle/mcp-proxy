// Package httpserver exposes HTTP frontends for proxied MCP processes.
//
// It provides:
//   - a control-plane server for health, process lifecycle, events, and RPC
//   - optional dedicated per-process RPC listeners bound to individual ports
//
// Both server types enforce timeout/header hardening and optional API-key
// authentication on non-health routes.
package httpserver
