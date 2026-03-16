package httpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/config"
	"github.com/vjanelle/mcp-proxy/internal/proxy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	It("serves health, process status, events, and RPC routes", func() {
		manager := helperManager()
		Expect(manager.Start("fake")).To(Succeed())
		defer manager.StopAll()

		srv := New("127.0.0.1:0", manager, SecurityOptions{})
		handler := srv.server.Handler

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
		handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(ContainSubstring(`"name":"fake"`))

		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
		for _, path := range []string{"/v1/processes/fake/rpc", "/mcp/fake"} {
			rec = httptest.NewRecorder()
			req = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
			handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK), path)
			Expect(rec.Body.String()).To(ContainSubstring(`"echo":"tools/list"`), path)
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/v1/processes/fake/rpc", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)))
		handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusAccepted))

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/v1/events?process=fake", nil)
		handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("returns bad request for missing process lifecycle commands", func() {
		srv := New("127.0.0.1:0", proxy.NewManager(nil), SecurityOptions{})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/processes/missing/start", nil)
		srv.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusBadRequest))
	})

	It("rejects unsupported methods and unknown routes", func() {
		srv := New("127.0.0.1:0", proxy.NewManager(nil), SecurityOptions{})
		handler := srv.server.Handler

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
		Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes/missing/rpc", nil))
		Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes/missing/start", nil))
		Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes/x", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))

		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp", nil))
		Expect(rec.Code).To(Equal(http.StatusTemporaryRedirect))
	})

	It("provides process-server health and can shutdown after start", func() {
		manager := proxy.NewManager(nil)
		ps := NewProcessServer("127.0.0.1:0", "fake", manager, SecurityOptions{})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		ps.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		s := New("127.0.0.1:0", manager, SecurityOptions{})
		done := make(chan error, 1)
		go func() {
			done <- s.Start()
		}()
		time.Sleep(50 * time.Millisecond)
		Expect(s.Shutdown(context.Background())).To(Succeed())
	})

	It("handles process-server rpc method errors", func() {
		ps := NewProcessServer("127.0.0.1:0", "missing", proxy.NewManager(nil), SecurityOptions{})

		rec := httptest.NewRecorder()
		ps.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rpc", nil))
		Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))

		rec = httptest.NewRecorder()
		ps.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"x"}`))))
		Expect(rec.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns accepted for process-server notifications", func() {
		manager := helperManager()
		Expect(manager.Start("fake")).To(Succeed())
		defer manager.StopAll()

		ps := NewProcessServer("127.0.0.1:0", "fake", manager, SecurityOptions{})
		rec := httptest.NewRecorder()
		ps.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))))
		Expect(rec.Code).To(Equal(http.StatusAccepted))
	})

	It("configures hardened http.Server timeouts and limits", func() {
		manager := proxy.NewManager(nil)
		srv := New("127.0.0.1:0", manager, SecurityOptions{})
		ps := NewProcessServer("127.0.0.1:0", "fake", manager, SecurityOptions{})

		Expect(srv.server.ReadHeaderTimeout).To(Equal(defaultReadHeaderTimeout))
		Expect(srv.server.ReadTimeout).To(Equal(defaultReadTimeout))
		Expect(srv.server.WriteTimeout).To(Equal(defaultWriteTimeout))
		Expect(srv.server.IdleTimeout).To(Equal(defaultIdleTimeout))
		Expect(srv.server.MaxHeaderBytes).To(Equal(defaultMaxHeaderBytes))

		Expect(ps.server.ReadHeaderTimeout).To(Equal(defaultReadHeaderTimeout))
		Expect(ps.server.ReadTimeout).To(Equal(defaultReadTimeout))
		Expect(ps.server.WriteTimeout).To(Equal(defaultWriteTimeout))
		Expect(ps.server.IdleTimeout).To(Equal(defaultIdleTimeout))
		Expect(ps.server.MaxHeaderBytes).To(Equal(defaultMaxHeaderBytes))
	})

	It("enforces request body size limits for RPC endpoints", func() {
		manager := helperManager()
		Expect(manager.Start("fake")).To(Succeed())
		defer manager.StopAll()

		opts := SecurityOptions{MaxRPCBodyBytes: 16}
		srv := New("127.0.0.1:0", manager, opts)
		ps := NewProcessServer("127.0.0.1:0", "fake", manager, opts)
		largePayload := bytes.Repeat([]byte("x"), 64)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/processes/fake/rpc", bytes.NewReader(largePayload))
		srv.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusRequestEntityTooLarge))

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(largePayload))
		ps.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusRequestEntityTooLarge))
	})

	It("requires an API key on non-health routes when configured", func() {
		manager := helperManager()
		Expect(manager.Start("fake")).To(Succeed())
		defer manager.StopAll()

		opts := SecurityOptions{APIKey: "secret-key"}
		srv := New("127.0.0.1:0", manager, opts)
		ps := NewProcessServer("127.0.0.1:0", "fake", manager, opts)
		payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)

		rec := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec = httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/processes", nil))
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))

		req := httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
		req.Header.Set(apiKeyHeader, "secret-key")
		rec = httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))

		rec = httptest.NewRecorder()
		ps.server.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(payload)))
		Expect(rec.Code).To(Equal(http.StatusUnauthorized))

		req = httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(payload))
		req.Header.Set(apiKeyHeader, "secret-key")
		rec = httptest.NewRecorder()
		ps.server.Handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("splits paths safely", func() {
		Expect(splitPath("/v1/processes/fake/rpc")).To(Equal([]string{"v1", "processes", "fake", "rpc"}))
		Expect(splitPath("/")).To(BeEmpty())
	})
})

func helperManager() *proxy.Manager {
	return proxy.NewManager([]config.ProcessConfig{
		{
			Name:      "fake",
			Command:   os.Args[0],
			Args:      []string{"-test.run=TestHelperProcessHTTP", "--"},
			Env:       append(os.Environ(), "GO_WANT_HTTP_HELPER=1"),
			Transport: "newline",
		},
	})
}

func TestHelperProcessHTTP(testingT *testing.T) {
	if os.Getenv("GO_WANT_HTTP_HELPER") != "1" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			os.Exit(0)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		msg := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(trimmed), &msg); err != nil {
			os.Exit(1)
		}

		idRaw, hasID := msg["id"]
		if !hasID {
			continue
		}

		method := ""
		if methodRaw, ok := msg["method"]; ok {
			_ = json.Unmarshal(methodRaw, &method)
		}

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(idRaw),
			"result":  map[string]string{"echo": method},
		}
		encoded, err := json.Marshal(resp)
		if err != nil {
			os.Exit(1)
		}

		_, _ = os.Stdout.Write(append(encoded, '\n'))
	}
}
