package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	It("routes JSON-RPC requests to a managed stdio process", func() {
		manager := NewManager([]config.ProcessConfig{
			{
				Name:    "fake",
				Command: os.Args[0],
				Args:    []string{"-test.run=TestHelperProcess", "--"},
				Env:     append(os.Environ(), "GO_WANT_HELPER_PROCESS=1"),
			},
		})

		Expect(manager.Start("fake")).To(Succeed())
		defer manager.StopAll()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		response, err := manager.DoRPC(ctx, "fake", []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{}}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(MatchJSON(`{"jsonrpc":"2.0","id":7,"result":{"echo":"tools/list"}}`))
	})
})

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		payload, err := readFrame(reader)
		if err != nil {
			os.Exit(0)
		}

		msg := map[string]json.RawMessage{}
		if unmarshalErr := json.Unmarshal(payload, &msg); unmarshalErr != nil {
			os.Exit(1)
		}

		id := msg["id"]
		method := msg["method"]
		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"result": map[string]any{
				"echo": string(method[1 : len(method)-1]),
			},
		}

		encoded, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			os.Exit(1)
		}

		if writeErr := writeFrame(os.Stdout, encoded); writeErr != nil {
			os.Exit(1)
		}
	}
}
