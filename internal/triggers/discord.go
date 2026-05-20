package triggers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

type discordTrigger struct {
	tenantID     string
	workflowName string
	version      string
	def          asmtypes.TriggerDef
	store        store.Store
	secretMgr    secrets.SecretManager
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func newDiscordTrigger(tenantID, workflowName, version string, def asmtypes.TriggerDef, s store.Store, sm secrets.SecretManager) (TriggerInstance, error) {
	return &discordTrigger{
		tenantID:     tenantID,
		workflowName: workflowName,
		version:      version,
		def:          def,
		store:        s,
		secretMgr:    sm,
	}, nil
}

func (d *discordTrigger) Start(ctx context.Context, startFn StartWorkflowFn) error {
	ctx, d.cancel = context.WithCancel(ctx)

	botTokenKey, ok := d.def.Config["bot_token_env"].(string)
	if !ok {
		botTokenKey = "DISCORD_BOT_TOKEN"
	}

	botToken, err := d.secretMgr.GetSecret(ctx, d.tenantID, botTokenKey)
	if err != nil || botToken == "" {
		return fmt.Errorf("discord bot token '%s' not found in tenant secrets", botTokenKey)
	}

	d.wg.Add(1)
	go d.connectLoop(ctx, botToken, startFn)
	return nil
}

func (d *discordTrigger) Stop(ctx context.Context) error {
	if d.cancel != nil {
		d.cancel()
		d.wg.Wait()
	}
	return nil
}

func (d *discordTrigger) connectLoop(ctx context.Context, token string, startFn StartWorkflowFn) {
	defer d.wg.Done()
	
	url := "wss://gateway.discord.gg/?v=10&encoding=json"
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		d.runSession(ctx, url, token, startFn)
		
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			// reconnect delay
		}
	}
}

type discordMessage struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  *int            `json:"s,omitempty"`
	T  *string         `json:"t,omitempty"`
}

func (d *discordTrigger) runSession(ctx context.Context, url, token string, startFn StartWorkflowFn) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("Discord gateway dial error", "error", err)
		}
		return
	}
	defer conn.Close()

	var sequence int
	var seqMu sync.Mutex

	// Read Hello
	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		slog.Warn("Discord gateway hello read error", "error", err)
		return
	}

	var hello discordMessage
	if err := json.Unmarshal(msgBytes, &hello); err != nil {
		slog.Warn("Discord hello parse error", "error", err)
		return
	}
	if hello.Op != 10 {
		slog.Warn("Expected op 10 hello", "op", hello.Op)
		return
	}

	var helloData struct {
		HeartbeatInterval int `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(hello.D, &helloData); err != nil {
		slog.Warn("Discord hello data parse error", "error", err)
		return
	}

	// Send Identify
	identify := map[string]interface{}{
		"op": 2,
		"d": map[string]interface{}{
			"token":   token,
			"intents": 512 | 32768, // GUILD_MESSAGES | MESSAGE_CONTENT
			"properties": map[string]interface{}{
				"os":      "linux",
				"browser": "asm",
				"device":  "asm",
			},
		},
	}
	if err := conn.WriteJSON(identify); err != nil {
		slog.Warn("Discord gateway identify error", "error", err)
		return
	}

	// Start heartbeater
	ctxHeartbeat, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTicker(time.Duration(helloData.HeartbeatInterval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctxHeartbeat.Done():
				return
			case <-ticker.C:
				seqMu.Lock()
				s := sequence
				seqMu.Unlock()
				
				hb := map[string]interface{}{
					"op": 1,
					"d":  nil,
				}
				if s > 0 {
					hb["d"] = s
				}
				if err := conn.WriteJSON(hb); err != nil {
					return // let read loop detect disconnect
				}
			}
		}
	}()

	// Wait for context cancellation to close the connection properly
	go func() {
		<-ctxHeartbeat.Done()
		conn.Close() // this breaks the read loop below
	}()

	// Read loop
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("Discord read error", "error", err)
			}
			return
		}

		var m discordMessage
		if err := json.Unmarshal(msgBytes, &m); err != nil {
			continue
		}

		if m.S != nil {
			seqMu.Lock()
			sequence = *m.S
			seqMu.Unlock()
		}

		if m.Op == 0 && m.T != nil && *m.T == "MESSAGE_CREATE" {
			var evt struct {
				Content   string `json:"content"`
				ChannelID string `json:"channel_id"`
				Author    struct {
					Username string `json:"username"`
					Bot      bool   `json:"bot"`
				} `json:"author"`
			}
			if err := json.Unmarshal(m.D, &evt); err == nil && !evt.Author.Bot && evt.Content != "" {
				inputs := map[string]interface{}{
					"channel_id":   evt.ChannelID,
					"message_text": evt.Content,
					"username":     evt.Author.Username,
				}
				go func(payload map[string]interface{}) {
					_, startErr := startFn(context.Background(), d.tenantID, d.workflowName, d.version, payload)
					if startErr != nil {
						slog.Error("Failed to start workflow from discord trigger", "workflow", d.workflowName, "error", startErr)
					} else {
						slog.Info("Started workflow from discord trigger", "workflow", d.workflowName)
					}
				}(inputs)
			}
		}
	}
}
