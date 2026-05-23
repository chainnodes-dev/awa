package triggers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

type telegramTrigger struct {
	tenantID     string
	workflowName string
	version      string
	def          asmtypes.TriggerDef
	store        store.Store
	secretMgr    secrets.SecretManager
	cancel       context.CancelFunc
	done         chan struct{}
}

func newTelegramTrigger(tenantID, workflowName, version string, def asmtypes.TriggerDef, s store.Store, sm secrets.SecretManager) (TriggerInstance, error) {
	return &telegramTrigger{
		tenantID:     tenantID,
		workflowName: workflowName,
		version:      version,
		def:          def,
		store:        s,
		secretMgr:    sm,
		done:         make(chan struct{}),
	}, nil
}

func (t *telegramTrigger) Start(ctx context.Context, startFn StartWorkflowFn) error {
	ctx, t.cancel = context.WithCancel(ctx)
	
	botTokenKey, ok := t.def.Config["bot_token_env"].(string)
	if !ok {
		botTokenKey = "TELEGRAM_BOT_TOKEN"
	}

	botToken, err := t.secretMgr.GetSecret(ctx, t.tenantID, botTokenKey)
	if err != nil || botToken == "" {
		return fmt.Errorf("telegram bot token '%s' not found in tenant secrets", botTokenKey)
	}

	go t.poll(ctx, botToken, startFn)
	return nil
}

func (t *telegramTrigger) Stop(ctx context.Context) error {
	if t.cancel != nil {
		t.cancel()
		<-t.done
	}
	return nil
}

func (t *telegramTrigger) poll(ctx context.Context, token string, startFn StartWorkflowFn) {
	defer close(t.done)
	
	offset := 0
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", token)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		reqBody, _ := json.Marshal(map[string]interface{}{
			"offset":  offset,
			"timeout": 20, // Long polling
		})

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// Expected context cancellation on stop
			if ctx.Err() != nil {
				return
			}
			slog.Warn("Telegram polling error", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var result struct {
			Ok     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  struct {
					MessageID int `json:"message_id"`
					Chat      struct {
						ID int64 `json:"id"`
					} `json:"chat"`
					Text string `json:"text"`
					From struct {
						Username string `json:"username"`
					} `json:"from"`
				} `json:"message"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		for _, update := range result.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			if update.Message.Text != "" {
				inputs := map[string]interface{}{
					"chat_id":      fmt.Sprintf("%d", update.Message.Chat.ID),
					"message_text": update.Message.Text,
					"username":     update.Message.From.Username,
				}

				go func(payload map[string]interface{}) {
					_, startErr := startFn(context.Background(), t.tenantID, t.workflowName, t.version, payload)
					if startErr != nil {
						slog.Error("Failed to start workflow from telegram trigger", "workflow", t.workflowName, "error", startErr)
					} else {
						slog.Info("Started workflow from telegram trigger", "workflow", t.workflowName)
					}
				}(inputs)
			}
		}
	}
}
