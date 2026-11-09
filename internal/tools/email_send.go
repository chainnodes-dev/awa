package tools

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EmailSendTool allows LLM agents and action steps to dispatch outbound emails via SMTP or cloud relay.
type EmailSendTool struct {
	defaultHost string
	defaultPort int
	defaultUser string
	defaultPass string
	defaultFrom string
}

func NewEmailSendTool() *EmailSendTool {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	return &EmailSendTool{
		defaultHost: os.Getenv("SMTP_HOST"),
		defaultPort: port,
		defaultUser: os.Getenv("SMTP_USER"),
		defaultPass: os.Getenv("SMTP_PASSWORD"),
		defaultFrom: os.Getenv("SMTP_FROM"),
	}
}

func (t *EmailSendTool) Name() string {
	return "email:send"
}

func (t *EmailSendTool) Description() string {
	return "Sends an outbound email to one or more recipients with subject, body, and optional HTML formatting."
}

func (t *EmailSendTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"to": {
				"type": "array",
				"items": { "type": "string" },
				"description": "Recipient email addresses (e.g. ['candidate@example.com'])"
			},
			"subject": {
				"type": "string",
				"description": "Email subject line"
			},
			"body": {
				"type": "string",
				"description": "Plain text email body"
			},
			"html": {
				"type": "string",
				"description": "Optional rich HTML email body"
			},
			"cc": {
				"type": "array",
				"items": { "type": "string" },
				"description": "Optional CC email addresses"
			}
		},
		"required": ["to", "subject", "body"]
	}`)
}

type emailSendInput struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	HTML    string   `json:"html,omitempty"`
	CC      []string `json:"cc,omitempty"`
}

func (t *EmailSendTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in emailSendInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("invalid input payload: %w", err)
	}

	if len(in.To) == 0 {
		return nil, fmt.Errorf("'to' must have at least one recipient")
	}
	if in.Subject == "" {
		return nil, fmt.Errorf("'subject' is required")
	}

	host := t.defaultHost
	port := t.defaultPort
	user := t.defaultUser
	pass := t.defaultPass
	from := t.defaultFrom
	if from == "" {
		from = user
	}
	if from == "" {
		from = "no-reply@asm-platform.internal"
	}

	messageID := fmt.Sprintf("<%s@asm-platform>", uuid.NewString())

	// If SMTP host is configured, attempt real SMTP transmission
	if host != "" && user != "" && pass != "" {
		addr := fmt.Sprintf("%s:%d", host, port)
		auth := smtp.PlainAuth("", user, pass, host)

		var msgBuilder strings.Builder
		msgBuilder.WriteString(fmt.Sprintf("From: %s\r\n", from))
		msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(in.To, ", ")))
		if len(in.CC) > 0 {
			msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(in.CC, ", ")))
		}
		msgBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", in.Subject))
		msgBuilder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
		msgBuilder.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
		msgBuilder.WriteString("MIME-Version: 1.0\r\n")

		if in.HTML != "" {
			msgBuilder.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
			msgBuilder.WriteString(in.HTML)
		} else {
			msgBuilder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
			msgBuilder.WriteString(in.Body)
		}

		allRecipients := append([]string{}, in.To...)
		allRecipients = append(allRecipients, in.CC...)

		var sendErr error
		if port == 465 {
			// SMTPS (Implicit TLS)
			tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
			if err != nil {
				return nil, fmt.Errorf("tls dial smtp: %w", err)
			}
			defer tlsConn.Close()

			c, err := smtp.NewClient(tlsConn, host)
			if err != nil {
				return nil, fmt.Errorf("smtp client: %w", err)
			}
			defer c.Quit()

			if err := c.Auth(auth); err != nil {
				return nil, fmt.Errorf("smtp auth: %w", err)
			}
			if err := c.Mail(from); err != nil {
				return nil, fmt.Errorf("smtp mail from: %w", err)
			}
			for _, rcpt := range allRecipients {
				if err := c.Rcpt(rcpt); err != nil {
					return nil, fmt.Errorf("smtp rcpt to %s: %w", rcpt, err)
				}
			}
			w, err := c.Data()
			if err != nil {
				return nil, fmt.Errorf("smtp data: %w", err)
			}
			_, _ = w.Write([]byte(msgBuilder.String()))
			_ = w.Close()
		} else {
			// Standard STARTTLS / Plain SMTP (port 587 or 25)
			conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
			if err != nil {
				return nil, fmt.Errorf("dial smtp %s: %w", addr, err)
			}
			defer conn.Close()

			client, err := smtp.NewClient(conn, host)
			if err != nil {
				return nil, fmt.Errorf("smtp client: %w", err)
			}
			defer client.Quit()

			if ok, _ := client.Extension("STARTTLS"); ok {
				if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
					return nil, fmt.Errorf("starttls: %w", err)
				}
			}
			if err := client.Auth(auth); err != nil {
				return nil, fmt.Errorf("smtp auth: %w", err)
			}
			sendErr = smtp.SendMail(addr, auth, from, allRecipients, []byte(msgBuilder.String()))
		}

		if sendErr != nil {
			return nil, fmt.Errorf("failed to send smtp email: %w", sendErr)
		}

		resBytes, _ := json.Marshal(map[string]interface{}{
			"status":     "sent",
			"to":         in.To,
			"subject":    in.Subject,
			"message_id": messageID,
			"relay":      "smtp",
			"timestamp":  time.Now().Format(time.RFC3339),
		})
		return resBytes, nil
	}

	// Simulation / dry-run mode when SMTP credentials are not yet configured
	resBytes, _ := json.Marshal(map[string]interface{}{
		"status":     "simulated_sent",
		"to":         in.To,
		"subject":    in.Subject,
		"message_id": messageID,
		"notice":     "Email accepted and simulated (configure SMTP_HOST, SMTP_USER, SMTP_PASSWORD to deliver live)",
		"timestamp":  time.Now().Format(time.RFC3339),
	})
	return resBytes, nil
}

