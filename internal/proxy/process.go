package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vjanelle/mcp-proxy/internal/config"
)

const (
	eventLifecycle = "lifecycle"
	eventStdErr    = "stderr"
	eventStdOut    = "stdout"
	eventRequest   = "request"
	eventResponse  = "response"
	eventProxyErr  = "proxy_error"
)

type managedProcess struct {
	cfg config.ProcessConfig

	mu      sync.RWMutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	running bool

	startedAt time.Time
	stoppedAt time.Time
	restarts  int
	lastError string

	pending map[string]chan []byte

	requests      atomic.Int64
	responses     atomic.Int64
	notifications atomic.Int64

	onEvent func(Event)
}

func newManagedProcess(cfg config.ProcessConfig, onEvent func(Event)) *managedProcess {
	return &managedProcess{
		cfg:     cfg,
		pending: map[string]chan []byte{},
		onEvent: onEvent,
	}
}

func (p *managedProcess) start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	cmd := exec.Command(p.cfg.Command, p.cfg.Args...) // #nosec G204 - configured command by local operator.
	if p.cfg.WorkDir != "" {
		cmd.Dir = p.cfg.WorkDir
	}

	if len(p.cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), p.cfg.Env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.running = true
	p.startedAt = time.Now()
	p.lastError = ""

	p.emitEvent(eventLifecycle, fmt.Sprintf("started pid=%d", cmd.Process.Pid))
	go p.readStdout(stdout)
	go p.readStderr(stderr)
	go p.waitForExit(cmd)

	return nil
}

func (p *managedProcess) stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill process: %w", err)
	}

	p.running = false
	p.stoppedAt = time.Now()
	p.emitEvent(eventLifecycle, "stop requested")

	return nil
}

func (p *managedProcess) restart() error {
	if err := p.stop(); err != nil {
		return err
	}

	p.mu.Lock()
	p.restarts++
	p.mu.Unlock()

	return p.start()
}

func (p *managedProcess) doRPC(ctx context.Context, payload []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("rpc context canceled: %w", err)
	}

	id, hasID, err := requestID(payload)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if !p.running || p.stdin == nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("process %q is not running", p.cfg.Name)
	}

	var responseCh chan []byte
	if hasID {
		responseCh = make(chan []byte, 1)
		p.pending[id] = responseCh
		p.requests.Add(1)
	} else {
		p.notifications.Add(1)
	}

	if err := p.writePayload(payload); err != nil {
		if hasID {
			delete(p.pending, id)
		}

		p.mu.Unlock()
		return nil, fmt.Errorf("write rpc to process: %w", err)
	}

	p.mu.Unlock()
	p.emitEvent(eventRequest, strings.TrimSpace(string(payload)))

	if !hasID {
		return nil, nil
	}

	select {
	case response := <-responseCh:
		return response, nil
	case <-ctx.Done():
		p.mu.Lock()
		delete(p.pending, id)
		p.mu.Unlock()
		return nil, fmt.Errorf("waiting for rpc response: %w", ctx.Err())
	}
}

func (p *managedProcess) status() ProcessStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pid := 0
	if p.cmd != nil && p.cmd.Process != nil {
		pid = p.cmd.Process.Pid
	}

	return ProcessStatus{
		Name:          p.cfg.Name,
		Port:          p.cfg.Port,
		Command:       strings.Join(append([]string{p.cfg.Command}, p.cfg.Args...), " "),
		Running:       p.running,
		PID:           pid,
		Restarts:      p.restarts,
		LastError:     p.lastError,
		StartedAt:     p.startedAt,
		StoppedAt:     p.stoppedAt,
		Requests:      p.requests.Load(),
		Responses:     p.responses.Load(),
		Notifications: p.notifications.Load(),
	}
}

func (p *managedProcess) readStdout(stdout io.Reader) {
	reader := bufio.NewReader(stdout)
	for {
		payload, err := p.readPayload(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				p.emitEvent(eventProxyErr, fmt.Sprintf("stdout read error: %v", err))
			}

			return
		}

		p.emitEvent(eventStdOut, strings.TrimSpace(string(payload)))
		p.routeResponse(payload)
	}
}

func (p *managedProcess) readPayload(reader *bufio.Reader) ([]byte, error) {
	if p.cfg.Transport == "newline" {
		return readNewlineFrame(reader)
	}

	return readFrame(reader)
}

func (p *managedProcess) writePayload(payload []byte) error {
	if p.cfg.Transport == "newline" {
		return writeNewlineFrame(p.stdin, payload)
	}

	return writeFrame(p.stdin, payload)
}

func (p *managedProcess) routeResponse(payload []byte) {
	id, hasID, err := requestID(payload)
	if err != nil || !hasID {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	waiter, ok := p.pending[id]
	if !ok {
		return
	}

	delete(p.pending, id)
	p.responses.Add(1)
	waiter <- payload
	close(waiter)
	p.emitEvent(eventResponse, strings.TrimSpace(string(payload)))
}

func (p *managedProcess) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		p.emitEvent(eventStdErr, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		p.emitEvent(eventProxyErr, fmt.Sprintf("stderr scan error: %v", err))
	}
}

func (p *managedProcess) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = false
	p.stoppedAt = time.Now()

	if err != nil {
		p.lastError = err.Error()
		p.emitEvent(eventLifecycle, fmt.Sprintf("exited with error: %v", err))
		return
	}

	p.emitEvent(eventLifecycle, "exited cleanly")
}

func (p *managedProcess) emitEvent(kind string, message string) {
	if p.onEvent == nil {
		return
	}

	p.onEvent(Event{
		Time:    time.Now(),
		Process: p.cfg.Name,
		Type:    kind,
		Message: message,
	})
}
