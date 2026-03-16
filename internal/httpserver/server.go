package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/proxy"
)

type Server struct {
	manager  *proxy.Manager
	security SecurityOptions
	server   *http.Server
}

// New builds the main control-plane HTTP server bound to addr.
func New(addr string, manager *proxy.Manager, security SecurityOptions) *Server {
	srv := &Server{
		manager:  manager,
		security: security.normalized(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.health)
	mux.HandleFunc("/v1/processes", requireAPIKey(srv.security.APIKey, srv.processes))
	mux.HandleFunc("/v1/processes/", requireAPIKey(srv.security.APIKey, srv.processAction))
	mux.HandleFunc("/v1/events", requireAPIKey(srv.security.APIKey, srv.events))
	mux.HandleFunc("/mcp/", requireAPIKey(srv.security.APIKey, srv.mcpAlias))

	srv.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	return srv
}

// Start runs the server until shutdown or listener error.
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.server.Shutdown(ctx)
}

func (s *Server) health(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) processes(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"processes": s.manager.List(),
	})
}

func (s *Server) events(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	processName := request.URL.Query().Get("process")
	events := s.manager.EventSnapshot(processName, 200)
	writeJSON(writer, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) processAction(writer http.ResponseWriter, request *http.Request) {
	parts := splitPath(request.URL.Path)
	if len(parts) < 4 {
		http.NotFound(writer, request)
		return
	}

	name := parts[2]
	action := parts[3]

	switch action {
	case "start":
		s.handleLifecycle(writer, request, name, s.manager.Start)
	case "stop":
		s.handleLifecycle(writer, request, name, s.manager.Stop)
	case "restart":
		s.handleLifecycle(writer, request, name, s.manager.Restart)
	case "rpc":
		s.rpc(writer, request, name)
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) mcpAlias(writer http.ResponseWriter, request *http.Request) {
	parts := splitPath(request.URL.Path)
	if len(parts) != 2 {
		http.NotFound(writer, request)
		return
	}

	name := parts[1]
	s.rpc(writer, request, name)
}

func (s *Server) handleLifecycle(
	writer http.ResponseWriter,
	request *http.Request,
	name string,
	fn func(string) error,
) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := fn(name); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ok",
		"name":   name,
	})
}

func (s *Server) rpc(writer http.ResponseWriter, request *http.Request, name string) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(writer, request.Body, s.security.MaxRPCBodyBytes)
	defer body.Close()
	payload, err := io.ReadAll(body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(writer, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(writer, fmt.Sprintf("read payload: %v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 60*time.Second)
	defer cancel()

	response, err := s.manager.DoRPC(ctx, name, payload)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if response == nil {
		writer.WriteHeader(http.StatusAccepted)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(response)
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}

	return strings.Split(trimmed, "/")
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}
