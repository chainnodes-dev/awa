package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/asm-platform/asm/internal/store"
)

// Manager handles the lifecycle of stdio-based MCP servers.
type Manager struct {
	store store.Store
	procs map[string]*stdioProcess
	mu    sync.Mutex
}

func NewManager(s store.Store) *Manager {
	return &Manager{
		store: s,
		procs: make(map[string]*stdioProcess),
	}
}

func (m *Manager) GetClient(ctx context.Context, name string) (Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.procs[name]; ok {
		return p, nil
	}

	srv, err := m.findServer(ctx, name)
	if err != nil {
		return nil, err
	}

	if srv.Transport != "stdio" {
		return NewSSEClient(srv.URL), nil
	}

	p, _, err := m.startStdioClient(srv)
	if err != nil {
		return nil, err
	}

	m.procs[name] = p
	return p, nil
}

func (m *Manager) findServer(ctx context.Context, name string) (*store.MCPServer, error) {
	servers, err := m.store.ListMCPServers(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range servers {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("MCP server %q not found", name)
}

func (m *Manager) startStdioClient(srv *store.MCPServer) (*stdioProcess, io.Closer, error) {
	cmd := exec.Command(srv.Command, srv.Args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	// Inject environment variables
	if len(srv.EnvVars) > 0 {
		env := os.Environ()
		for k, v := range srv.EnvVars {
			if v == "" {
				v = os.Getenv(k)
			}
			if v != "" {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start mcp server: %w", err)
	}

	p := &stdioProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		encoder: json.NewEncoder(stdin),
		decoder: json.NewDecoder(stdout),
		stderr:  stderr,
	}
	return p, p, nil
}

// TestStdioClientMap starts a temporary process for discovery/testing with multiple env vars.
func (m *Manager) TestStdioClientMap(command string, args []string, envVars map[string]string) (Client, io.Closer, error) {
	cmd := exec.Command(command, args...)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if len(envVars) > 0 {
		env := os.Environ()
		for k, v := range envVars {
			if v == "" {
				v = os.Getenv(k)
			}
			if v != "" {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start mcp server: %w (stderr: %s)", err, stderr.String())
	}

	p := &stdioProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		encoder: json.NewEncoder(stdin),
		decoder: json.NewDecoder(stdout),
		stderr:  stderr,
	}
	return p, p, nil
}

type stdioProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	encoder *json.Encoder
	decoder *json.Decoder
	stderr  *bytes.Buffer
	idCount int
	mu      sync.Mutex
}

func (p *stdioProcess) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.procs {
		slog.Info("Stopping MCP server", "id", id)
		_ = p.Close()
	}
	m.procs = make(map[string]*stdioProcess)
}

type Client interface {
	Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
	Notify(ctx context.Context, method string, params interface{}) error
}

func (p *stdioProcess) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	p.mu.Lock()
	p.idCount++
	id := p.idCount
	p.mu.Unlock()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := p.encoder.Encode(req); err != nil {
		return nil, err
	}

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := p.decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			// Check if process exited with an error
			if exitErr := p.cmd.Wait(); exitErr != nil {
				return nil, fmt.Errorf("mcp server exited: %v (stderr: %s)", exitErr, p.stderr.String())
			}
			return nil, fmt.Errorf("mcp server closed connection (EOF). stderr: %s", p.stderr.String())
		}
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("mcp error (%d): %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

func (p *stdioProcess) Notify(ctx context.Context, method string, params interface{}) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return p.encoder.Encode(req)
}

// sseClient implementation
type sseClient struct {
	url string
}

func NewSSEClient(url string) Client {
	return &sseClient{url: url}
}

func (c *sseClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Simple HTTP POST for SSE (assuming an SSE-to-RPC bridge or similar)
	// In Phaxa, SSE servers often handle JSON-RPC via POST to /rpc
	payload, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  method,
		"params":  params,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", c.url, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r struct {
		Result json.RawMessage `json:"result"`
		Error  interface{}     `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return r.Result, nil
}

func (c *sseClient) Notify(ctx context.Context, method string, params interface{}) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.url, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
