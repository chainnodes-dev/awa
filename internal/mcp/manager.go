package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/asm-platform/asm/internal/store"
)

// Manager handles the lifecycle of stdio-based MCP servers.
type Manager struct {
	store store.Store
	mu    sync.RWMutex
	procs map[string]*stdioProcess
}

func NewManager(s store.Store) *Manager {
	return &Manager{
		store: s,
		procs: make(map[string]*stdioProcess),
	}
}

type stdioProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	encoder *json.Encoder
	decoder *json.Decoder
	idCount int
}

// GetClient returns an MCP client for the given server.
// If the server is stdio-based, it ensures the process is running.
func (m *Manager) GetClient(ctx context.Context, serverName string) (Client, error) {
	// 1. Fetch server def from store
	servers, err := m.store.ListMCPServers(ctx)
	if err != nil {
		return nil, err
	}

	var target *store.MCPServer
	for _, s := range servers {
		if s.Name == serverName {
			target = s
			break
		}
	}

	if target == nil {
		return nil, fmt.Errorf("mcp server %q not found", serverName)
	}

	if target.Transport == "stdio" {
		return m.getStdioClient(target)
	}

	// Default to SSE client (existing logic)
	return NewSSEClient(target.URL), nil
}

func (m *Manager) getStdioClient(srv *store.MCPServer) (Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.procs[srv.ID]; ok {
		// Check if process is still alive
		if p.cmd.ProcessState == nil {
			return p, nil
		}
		// Process died, clean up
		delete(m.procs, srv.ID)
	}

	// Start new process
	slog.Info("Starting local MCP server", "name", srv.Name, "command", srv.Command, "args", srv.Args)
	cmd := exec.Command(srv.Command, srv.Args...)
	cmd.Stderr = os.Stderr
	
	// Inject environment variable if configured
	if srv.EnvVar != "" {
		val := srv.EnvValue
		if val == "" {
			val = os.Getenv(srv.EnvVar)
		}
		if val != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", srv.EnvVar, val))
		}
	}
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mcp server %q: %w", srv.Name, err)
	}

	p := &stdioProcess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		encoder: json.NewEncoder(stdin),
		decoder: json.NewDecoder(stdout),
	}
	m.procs[srv.ID] = p

	return p, nil
}

// TestStdioClient starts a temporary process for discovery/testing.
// The caller is responsible for calling Close() on the returned process.
func (m *Manager) TestStdioClient(command string, args []string, envVar, envValue string) (Client, io.Closer, error) {
	slog.Info("Starting temp local MCP server", "command", command)
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	
	if envVar != "" {
		val := envValue
		if val == "" {
			val = os.Getenv(envVar)
		}
		if val != "" {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", envVar, val))
		}
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
	}
	return p, p, nil
}

func (p *stdioProcess) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

// Shutdown kills all managed processes.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.procs {
		slog.Info("Stopping MCP server", "id", id)
		_ = p.Close()
	}
	m.procs = make(map[string]*stdioProcess)
}

// Client interface represents a connection to an MCP server.
type Client interface {
	Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
	Notify(ctx context.Context, method string, params interface{}) error
}

// stdioProcess implementation of Client
func (p *stdioProcess) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	p.idCount++
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      p.idCount,
		"method":  method,
		"params":  params,
	}
	
	if err := p.encoder.Encode(req); err != nil {
		return nil, err
	}

	var resp struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	// For the very first message after process start, we might want to skip non-JSON lines (like npm install progress)
	// But our decoder is persistent. Let's just catch the error and log it.
	if err := p.decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("mcp rpc decode error: %w (check server logs/stderr for details)", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("mcp error: %s", resp.Error.Message)
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

// NewSSEClient (Stub for existing logic)
func NewSSEClient(url string) Client {
	return &sseClient{url: url}
}

type sseClient struct {
	url         string
	endpointURL string
	responses   sync.Map // map[interface{}]chan json.RawMessage
	stopOnce    sync.Once
	stop        chan struct{}
	idCounter   int64
}

func (c *sseClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if c.endpointURL == "" {
		if err := c.startSession(ctx); err != nil {
			return nil, err
		}
	}

	id := atomic.AddInt64(&c.idCounter, 1)
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	
	respChan := make(chan json.RawMessage, 1)
	c.responses.Store(id, respChan)
	defer c.responses.Delete(id)

	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpointURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	postResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusOK && postResp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("POST failed: %s", postResp.Status)
	}

	// Wait for response from SSE stream
	select {
	case result := <-respChan:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.stop:
		return nil, fmt.Errorf("client closed")
	}
}

func (c *sseClient) Notify(ctx context.Context, method string, params interface{}) error {
	if c.endpointURL == "" {
		if err := c.startSession(ctx); err != nil {
			return err
		}
	}
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.endpointURL, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	_, err := http.DefaultClient.Do(req)
	return err
}

func (c *sseClient) Close() error {
	c.stopOnce.Do(func() {
		close(c.stop)
	})
	return nil
}

func (c *sseClient) startSession(ctx context.Context) error {
	c.stop = make(chan struct{})
	
	// First resolve the endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE GET failed: %s", resp.Status)
	}

	// Read first events to find endpoint
	scanner := bufio.NewScanner(resp.Body)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.ReplaceAll(line, " ", ""), "event:endpoint") {
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data:") {
					path := strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))
					if strings.HasPrefix(path, "http") {
						c.endpointURL = path
					} else {
						base, _ := url.Parse(c.url)
						rel, _ := url.Parse(path)
						c.endpointURL = base.ResolveReference(rel).String()
					}
					found = true
					break
				}
			}
		}
	}

	if !found {
		resp.Body.Close()
		return fmt.Errorf("could not find endpoint event")
	}

	// Start background listener for messages
	go func() {
		defer resp.Body.Close()
		for scanner.Scan() {
			select {
			case <-c.stop:
				return
			default:
			}
			line := scanner.Text()
			if strings.HasPrefix(strings.ReplaceAll(line, " ", ""), "event:message") {
				if scanner.Scan() {
					dataLine := scanner.Text()
					if strings.HasPrefix(dataLine, "data:") {
						content := strings.TrimPrefix(dataLine, "data:")
						var rpcResp struct {
							ID     interface{}     `json:"id"`
							Result json.RawMessage `json:"result"`
							Error  interface{}     `json:"error"`
						}
						if err := json.Unmarshal([]byte(content), &rpcResp); err == nil && rpcResp.ID != nil {
							// Handle numeric ID type mismatch (JSON unmarshals numbers as float64 into interface{})
							var lookupID interface{} = rpcResp.ID
							if f, ok := rpcResp.ID.(float64); ok {
								lookupID = int64(f)
							}
							
							slog.Debug("Received MCP SSE message", "id", lookupID)
							if ch, ok := c.responses.Load(lookupID); ok {
								ch.(chan json.RawMessage) <- rpcResp.Result
							}
						} else if err != nil {
							slog.Error("Failed to unmarshal MCP SSE message", "error", err, "content", content)
						}
					}
				}
			}
		}
	}()


	return nil
}
