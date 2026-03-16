package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	defaultListenAddr      = "127.0.0.1:18080"
	defaultMaxRPCBodyBytes = 1 << 20 // 1 MiB
)

// ProcessConfig describes one managed MCP process.
type ProcessConfig struct {
	// Name is the unique process identifier used in API routes and the TUI.
	Name string `json:"name"`
	// Command is the executable to launch.
	Command string `json:"command"`
	// Args are command line arguments passed to Command.
	Args []string `json:"args"`
	// WorkDir sets the process working directory when non-empty.
	WorkDir string `json:"workDir"`
	// Env appends KEY=VALUE environment entries to the child process.
	Env []string `json:"env"`
	// AutoStart launches the process automatically at proxy startup.
	AutoStart bool `json:"autoStart"`
	// Port enables a dedicated local HTTP listener for this process.
	Port int `json:"port"`
	// Transport selects stdio framing: "content-length" or "newline".
	Transport string `json:"transport"`
}

// Config is the top-level mcp-proxy configuration document.
type Config struct {
	// ListenAddr is the main control plane listener (for /v1 endpoints).
	ListenAddr string `json:"listenAddr"`
	// AllowRemote permits binding the control listener to non-loopback hosts.
	AllowRemote bool `json:"allowRemote"`
	// APIKey secures non-health HTTP endpoints when set.
	APIKey string `json:"apiKey"`
	// APIKeyEnv optionally names an environment variable holding APIKey.
	APIKeyEnv string `json:"apiKeyEnv"`
	// MaxRPCBodyBytes bounds proxied JSON-RPC request payload sizes.
	MaxRPCBodyBytes int64 `json:"maxRPCBodyBytes"`
	// Processes is the set of managed MCP stdio processes.
	Processes []ProcessConfig `json:"processes"`
}

// Load reads, validates, and normalizes a JSON config file from path.
// It applies defaults for missing ListenAddr and ProcessConfig.Transport.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Config{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}
	if cfg.MaxRPCBodyBytes <= 0 {
		cfg.MaxRPCBodyBytes = defaultMaxRPCBodyBytes
	}
	if strings.TrimSpace(cfg.APIKey) == "" && strings.TrimSpace(cfg.APIKeyEnv) != "" {
		cfg.APIKey = os.Getenv(strings.TrimSpace(cfg.APIKeyEnv))
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)

	for i := range cfg.Processes {
		if cfg.Processes[i].Name == "" {
			return Config{}, fmt.Errorf("process at index %d has empty name", i)
		}

		if cfg.Processes[i].Command == "" {
			return Config{}, fmt.Errorf("process %q has empty command", cfg.Processes[i].Name)
		}

		if cfg.Processes[i].Transport == "" {
			cfg.Processes[i].Transport = "content-length"
		}
	}
	if err := validateNetworkSecurity(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validateNetworkSecurity(cfg Config) error {
	host, _, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listenAddr must be host:port: %w", err)
	}
	if isLoopbackHost(host) {
		return nil
	}

	if !cfg.AllowRemote {
		return fmt.Errorf("listenAddr %q is non-loopback; set allowRemote=true to permit remote access", cfg.ListenAddr)
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("apiKey (or apiKeyEnv) is required when allowRemote=true on non-loopback listenAddr")
	}

	return nil
}

func isLoopbackHost(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return false
	}
	if strings.EqualFold(trimmed, "localhost") {
		return true
	}

	ip := net.ParseIP(trimmed)
	return ip != nil && ip.IsLoopback()
}
