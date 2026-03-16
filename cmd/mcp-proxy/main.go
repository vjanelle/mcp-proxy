package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/vjanelle/mcp-proxy/internal/config"
	"github.com/vjanelle/mcp-proxy/internal/httpserver"
	"github.com/vjanelle/mcp-proxy/internal/proxy"
	"github.com/vjanelle/mcp-proxy/internal/tui"
)

type programRunner func(*proxy.Manager) error

func main() {
	var configPath string
	var noTUI bool

	flag.StringVar(&configPath, "config", "mcp-proxy.json", "path to mcp proxy config json")
	flag.BoolVar(&noTUI, "no-tui", false, "disable the local debug TUI")
	flag.Parse()

	if err := runApp(configPath, noTUI, waitForSignal, defaultProgramRunner); err != nil {
		log.Fatalf("%v", err)
	}
}

func runApp(configPath string, noTUI bool, waitFn func(), runProgram programRunner) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	manager := proxy.NewManager(cfg.Processes)
	if err := manager.StartAutoProcesses(); err != nil {
		return fmt.Errorf("start auto processes: %w", err)
	}
	defer manager.StopAll()

	serverSecurity := httpserver.SecurityOptions{
		APIKey:          cfg.APIKey,
		MaxRPCBodyBytes: cfg.MaxRPCBodyBytes,
	}
	server := httpserver.New(cfg.ListenAddr, manager, serverSecurity)
	processServers := newProcessServers(cfg, manager)
	go func() {
		err := server.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server error: %v", err)
		}
	}()
	for _, ps := range processServers {
		processServer := ps
		go func() {
			err := processServer.Start()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("process server error: %v", err)
			}
		}()
	}

	log.Printf("mcp proxy listening at http://%s", cfg.ListenAddr)
	appCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	if noTUI {
		waitFn()
		shutdown(appCtx, server, processServers)
		return nil
	}

	if err := runProgram(manager); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}

	shutdown(appCtx, server, processServers)
	return nil
}

func defaultProgramRunner(manager *proxy.Manager) error {
	program := tea.NewProgram(tui.New(manager))
	if _, err := program.Run(); err != nil {
		return err
	}

	return nil
}

func waitForSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}

func shutdown(parent context.Context, server *httpserver.Server, processServers []*httpserver.ProcessServer) {
	serverCtx, cancelServer := context.WithTimeout(parent, 3*time.Second)
	defer cancelServer()
	if err := server.Shutdown(serverCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown server: %v\n", err)
	}

	for _, ps := range processServers {
		processCtx, cancelProcess := context.WithTimeout(parent, 3*time.Second)
		err := ps.Shutdown(processCtx)
		cancelProcess()
		if err != nil {
			fmt.Fprintf(os.Stderr, "shutdown process server: %v\n", err)
		}
	}
}

func newProcessServers(cfg config.Config, manager *proxy.Manager) []*httpserver.ProcessServer {
	serverSecurity := httpserver.SecurityOptions{
		APIKey:          cfg.APIKey,
		MaxRPCBodyBytes: cfg.MaxRPCBodyBytes,
	}

	host := "127.0.0.1"
	if parsedHost, _, err := net.SplitHostPort(cfg.ListenAddr); err == nil && parsedHost != "" {
		host = parsedHost
	}

	servers := make([]*httpserver.ProcessServer, 0, len(cfg.Processes))
	for _, process := range cfg.Processes {
		if process.Port <= 0 {
			continue
		}

		addr := fmt.Sprintf("%s:%d", host, process.Port)
		log.Printf("process %q rpc listening at http://%s/rpc", process.Name, addr)
		servers = append(servers, httpserver.NewProcessServer(addr, process.Name, manager, serverSecurity))
	}

	return servers
}
