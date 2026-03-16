package config_test

import (
	"os"
	"path/filepath"

	"github.com/vjanelle/mcp-proxy/internal/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Load", func() {
	It("applies defaults for listen address and transport", func() {
		path := writeConfig(`{"processes":[{"name":"p1","command":"echo"}]}`)
		cfg, err := config.Load(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ListenAddr).To(Equal("127.0.0.1:18080"))
		Expect(cfg.Processes[0].Transport).To(Equal("content-length"))
		Expect(cfg.MaxRPCBodyBytes).To(BeNumerically(">", 0))
	})

	It("rejects invalid process definitions", func() {
		for _, raw := range []string{
			`{"processes":[{"command":"echo"}]}`,
			`{"processes":[{"name":"p1"}]}`,
			`{`,
		} {
			_, err := config.Load(writeConfig(raw))
			Expect(err).To(HaveOccurred())
		}
	})

	It("rejects non-loopback listen addresses unless explicitly allowed", func() {
		_, err := config.Load(writeConfig(`{
			"listenAddr":"0.0.0.0:18080",
			"processes":[{"name":"p1","command":"echo"}]
		}`))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("allowRemote=true"))
	})

	It("requires API key when remote access is explicitly enabled", func() {
		_, err := config.Load(writeConfig(`{
			"listenAddr":"0.0.0.0:18080",
			"allowRemote":true,
			"processes":[{"name":"p1","command":"echo"}]
		}`))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("apiKey"))
	})

	It("supports resolving apiKey from apiKeyEnv", func() {
		Expect(os.Setenv("MCP_PROXY_TEST_API_KEY", "test-secret")).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("MCP_PROXY_TEST_API_KEY")).To(Succeed())
		})

		cfg, err := config.Load(writeConfig(`{
			"listenAddr":"0.0.0.0:18080",
			"allowRemote":true,
			"apiKeyEnv":"MCP_PROXY_TEST_API_KEY",
			"processes":[{"name":"p1","command":"echo"}]
		}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.APIKey).To(Equal("test-secret"))
	})
})

func writeConfig(raw string) string {
	dir, err := os.MkdirTemp("", "cfg-*")
	Expect(err).NotTo(HaveOccurred())

	path := filepath.Join(dir, "config.json")
	err = os.WriteFile(path, []byte(raw), 0o600)
	Expect(err).NotTo(HaveOccurred())

	return path
}
