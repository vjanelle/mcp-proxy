// Package proxy manages stdio MCP subprocesses and request/response routing.
//
// It is responsible for lifecycle control, JSON-RPC forwarding, transport
// framing, and structured status/event snapshots consumed by HTTP and TUI
// frontends.
package proxy
