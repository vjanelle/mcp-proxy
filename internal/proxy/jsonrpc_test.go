package proxy

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JSON-RPC helpers", func() {
	It("extracts request id and notification state", func() {
		id, hasID, err := requestID([]byte(`{"jsonrpc":"2.0","id":"abc","method":"x"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(hasID).To(BeTrue())
		Expect(id).To(Equal(`"abc"`))

		_, hasID, err = requestID([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(hasID).To(BeFalse())
	})

	It("rejects batch payloads", func() {
		_, _, err := requestID([]byte(`[{"jsonrpc":"2.0","id":1}]`))
		Expect(err).To(HaveOccurred())
	})

	It("supports newline frame roundtrip", func() {
		buf := bytes.NewBuffer(nil)
		payload := []byte(`{"jsonrpc":"2.0","id":1}`)
		Expect(writeNewlineFrame(buf, payload)).To(Succeed())
		got, err := readNewlineFrame(bufio.NewReader(bytes.NewReader(buf.Bytes())))
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(payload))
	})

	It("returns errors for unknown process operations and snapshots events", func() {
		manager := NewManager(nil)
		_, err := manager.DoRPC(context.Background(), "missing", []byte(`{}`))
		Expect(err).To(HaveOccurred())
		Expect(manager.Start("missing")).To(HaveOccurred())
		Expect(manager.Stop("missing")).To(HaveOccurred())
		Expect(manager.Restart("missing")).To(HaveOccurred())

		snap := manager.EventSnapshot("", 10)
		Expect(snap).To(BeEmpty())
	})

	It("starts autostart processes and supports restart/stop", func() {
		manager := NewManager([]config.ProcessConfig{
			{
				Name:      "auto",
				Command:   os.Args[0],
				Args:      []string{"-test.run=TestHelperProcess", "--"},
				Env:       append(os.Environ(), "GO_WANT_HELPER_PROCESS=1"),
				Transport: "content-length",
				AutoStart: true,
			},
		})
		Expect(manager.StartAutoProcesses()).To(Succeed())
		defer manager.StopAll()

		statuses := manager.List()
		Expect(statuses).To(HaveLen(1))
		Expect(statuses[0].Running).To(BeTrue())

		Expect(manager.Restart("auto")).To(Succeed())
		time.Sleep(20 * time.Millisecond)
		Expect(manager.Stop("auto")).To(Succeed())
	})

	It("returns timeout when process does not reply", func() {
		manager := NewManager([]config.ProcessConfig{
			{
				Name:      "silent",
				Command:   os.Args[0],
				Args:      []string{"-test.run=TestSilentHelperProcess", "--"},
				Env:       append(os.Environ(), "GO_WANT_SILENT_HELPER=1"),
				Transport: "newline",
			},
		})
		Expect(manager.Start("silent")).To(Succeed())
		defer manager.StopAll()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := manager.DoRPC(ctx, "silent", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
		Expect(err).To(HaveOccurred())
	})

	It("tracks notification counters", func() {
		manager := NewManager([]config.ProcessConfig{
			{
				Name:      "notify",
				Command:   os.Args[0],
				Args:      []string{"-test.run=TestSilentHelperProcess", "--"},
				Env:       append(os.Environ(), "GO_WANT_SILENT_HELPER=1"),
				Transport: "newline",
			},
		})
		Expect(manager.Start("notify")).To(Succeed())
		defer manager.StopAll()

		_, err := manager.DoRPC(context.Background(), "notify", []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
		Expect(err).NotTo(HaveOccurred())
		status := manager.List()[0]
		Expect(status.Notifications).To(Equal(int64(1)))
	})

	It("returns error for failed autostart command", func() {
		manager := NewManager([]config.ProcessConfig{
			{Name: "bad", Command: "definitely-not-a-command", AutoStart: true},
		})
		Expect(manager.StartAutoProcesses()).To(HaveOccurred())
	})
})

func TestSilentHelperProcess(testingT *testing.T) {
	if os.Getenv("GO_WANT_SILENT_HELPER") != "1" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// Intentionally no response; used to test timeout and notifications.
	}

	os.Exit(0)
}
