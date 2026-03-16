package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/proxy"
)

type ProcessServer struct {
	name     string
	manager  *proxy.Manager
	security SecurityOptions
	server   *http.Server
}

// NewProcessServer builds a dedicated RPC server for a single process name.
func NewProcessServer(addr string, name string, manager *proxy.Manager, security SecurityOptions) *ProcessServer {
	s := &ProcessServer{
		name:     name,
		manager:  manager,
		security: security.normalized(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/rpc", requireAPIKey(s.security.APIKey, s.rpc))
	mux.HandleFunc("/", requireAPIKey(s.security.APIKey, s.rpc))

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	return s
}

// Start runs the dedicated process server until shutdown or listener error.
func (s *ProcessServer) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the process server.
func (s *ProcessServer) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.server.Shutdown(ctx)
}

func (s *ProcessServer) health(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"status":  "ok",
		"process": s.name,
	})
}

func (s *ProcessServer) rpc(writer http.ResponseWriter, request *http.Request) {
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

	response, err := s.manager.DoRPC(ctx, s.name, payload)
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
