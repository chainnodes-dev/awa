package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/email"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/store"
)

// InboundEmailWebhook handles incoming webhook calls from cloud email gateways
// (SendGrid, Mailgun, Postmark, AWS SES, Cloudflare Email Routing) or raw RFC822 streams.
// POST /api/v1/webhooks/inbound-email
// POST /api/v1/webhooks/inbound-email/:tenant_id
func (h *Handlers) InboundEmailWebhook(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		tenantID = c.Query("tenant_id")
	}
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

	targetWorkflow := c.Query("workflow")
	contentType := c.GetHeader("Content-Type")

	var parsed *email.ParsedEmail
	var err error

	if strings.HasPrefix(contentType, "message/rfc822") {
		// 1. Raw RFC822 stream
		bodyBytes, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read raw rfc822 stream"})
			return
		}
		parsed, err = email.ParseRFC822(bytes.NewReader(bodyBytes))
	} else if strings.HasPrefix(contentType, "application/json") {
		// 2. Structured JSON payload
		var req struct {
			From        string `json:"from"`
			To          interface{} `json:"to"` // string or []string
			Cc          interface{} `json:"cc"`
			Subject     string `json:"subject"`
			Text        string `json:"text"`
			HTML        string `json:"html"`
			MessageID   string `json:"message_id"`
			Date        string `json:"date"`
			Attachments []struct {
				Name          string `json:"name"`
				ContentBase64 string `json:"content_base64"`
				MimeType      string `json:"mime_type"`
			} `json:"attachments"`
		}

		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + bindErr.Error()})
			return
		}

		var toList []string
		switch v := req.To.(type) {
		case string:
			if v != "" {
				toList = []string{v}
			}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					toList = append(toList, s)
				}
			}
		}

		msgID := req.MessageID
		if msgID == "" {
			msgID = uuid.NewString()
		}

		date := time.Now()
		if req.Date != "" {
			if t, parseErr := time.Parse(time.RFC3339, req.Date); parseErr == nil {
				date = t
			}
		}

		var atts []events.EmailAttachment
		for _, a := range req.Attachments {
			if a.Name != "" && a.ContentBase64 != "" {
				savedAtt, attErr := email.SaveBase64Attachment(a.Name, a.MimeType, a.ContentBase64)
				if attErr == nil {
					atts = append(atts, savedAtt)
				}
			}
		}

		parsed = &email.ParsedEmail{
			MessageID:   msgID,
			From:        req.From,
			To:          toList,
			Subject:     req.Subject,
			Date:        date,
			TextBody:    req.Text,
			HTMLBody:    req.HTML,
			Attachments: atts,
		}
	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		// 3. Multipart form data (SendGrid parse / Mailgun webhook)
		form, formErr := c.MultipartForm()
		if formErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form: " + formErr.Error()})
			return
		}

		// Check if raw RFC822 was posted in form field "email" or "message"
		if rawFields := form.Value["email"]; len(rawFields) > 0 && rawFields[0] != "" {
			parsed, err = email.ParseRFC822(strings.NewReader(rawFields[0]))
		} else if rawFields := form.Value["message"]; len(rawFields) > 0 && rawFields[0] != "" {
			parsed, err = email.ParseRFC822(strings.NewReader(rawFields[0]))
		} else {
			// Extract standard fields
			from := getFirstFormValue(form.Value, "from", "sender")
			toStr := getFirstFormValue(form.Value, "to", "recipient")
			subject := getFirstFormValue(form.Value, "subject")
			text := getFirstFormValue(form.Value, "text", "body-plain", "stripped-text")
			html := getFirstFormValue(form.Value, "html", "body-html", "stripped-html")
			msgID := getFirstFormValue(form.Value, "Message-Id", "message-id")
			if msgID == "" {
				msgID = uuid.NewString()
			}

			var toList []string
			if toStr != "" {
				toList = []string{toStr}
			}

			// Extract file attachments
			var atts []events.EmailAttachment
			for _, fileHeaders := range form.File {
				for _, fh := range fileHeaders {
					f, openErr := fh.Open()
					if openErr != nil {
						continue
					}
					data, readErr := io.ReadAll(f)
					_ = f.Close()
					if readErr != nil {
						continue
					}

					saved, saveErr := saveMultipartFile(fh.Filename, fh.Header.Get("Content-Type"), data)
					if saveErr == nil {
						atts = append(atts, saved)
					}
				}
			}

			parsed = &email.ParsedEmail{
				MessageID:   msgID,
				From:        from,
				To:          toList,
				Subject:     subject,
				Date:        time.Now(),
				TextBody:    text,
				HTMLBody:    html,
				Attachments: atts,
			}
		}
	} else {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "unsupported content-type: " + contentType})
		return
	}

	if err != nil || parsed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to parse email: %v", err)})
		return
	}

	// Dispatch ingestion via email service
	if h.emailService == nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":         true,
			"message_id": parsed.MessageID,
			"subject":    parsed.Subject,
			"warning":    "email service not initialized; email parsed but not dispatched",
		})
		return
	}

	run, ingestErr := h.emailService.IngestEmail(c.Request.Context(), tenantID, parsed, "webhook", targetWorkflow)
	if ingestErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest email: " + ingestErr.Error()})
		return
	}

	res := gin.H{
		"ok":                true,
		"message_id":        parsed.MessageID,
		"subject":           parsed.Subject,
		"from":              parsed.From,
		"attachments_count": len(parsed.Attachments),
	}
	if run != nil {
		res["run_id"] = run.ID
		res["workflow_name"] = run.WorkflowName
		res["workflow_status"] = run.Status
	}

	c.JSON(http.StatusOK, res)
}

func getFirstFormValue(values map[string][]string, keys ...string) string {
	for _, k := range keys {
		if vals, ok := values[k]; ok && len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return ""
}

func saveMultipartFile(filename, contentType string, data []byte) (events.EmailAttachment, error) {
	fileID := uuid.NewString()
	cleanName := filepath.Base(filename)
	if cleanName == "" || cleanName == "." {
		cleanName = "attachment.dat"
	}

	uploadDir := filepath.Join("data", "uploads", fileID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return events.EmailAttachment{}, err
	}

	savePath := filepath.Join(uploadDir, cleanName)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return events.EmailAttachment{}, err
	}

	return events.EmailAttachment{
		FileID:      fileID,
		Filename:    cleanName,
		ContentType: contentType,
		Size:        int64(len(data)),
		URL:         fmt.Sprintf("/api/v1/uploads/%s/%s", fileID, cleanName),
	}, nil
}
