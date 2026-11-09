package email

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

	"github.com/asm-platform/asm/internal/events"
)

// ParsedEmail represents a parsed inbound email message.
type ParsedEmail struct {
	MessageID   string                   `json:"message_id"`
	From        string                   `json:"from"`
	To          []string                 `json:"to"`
	Cc          []string                 `json:"cc,omitempty"`
	Subject     string                   `json:"subject"`
	Date        time.Time                `json:"date"`
	TextBody    string                   `json:"text_body"`
	HTMLBody    string                   `json:"html_body,omitempty"`
	Attachments []events.EmailAttachment `json:"attachments,omitempty"`
	Headers     map[string]string        `json:"headers,omitempty"`
}

var wordDecoder = mime.WordDecoder{
	CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	},
}

// DecodeHeader decodes RFC 2047 encoded-word strings (e.g. =?UTF-8?B?...?=).
func DecodeHeader(val string) string {
	decoded, err := wordDecoder.DecodeHeader(val)
	if err != nil {
		return val
	}
	return decoded
}

// ParseRFC822 parses raw RFC 5322 / RFC 822 email bytes into a structured ParsedEmail,
// storing any binary or document attachments to data/uploads/.
func ParseRFC822(r io.Reader) (*ParsedEmail, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, fmt.Errorf("read rfc822 message: %w", err)
	}

	headers := make(map[string]string)
	for k, v := range msg.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = DecodeHeader(v[0])
		}
	}

	from := DecodeHeader(msg.Header.Get("From"))
	subject := DecodeHeader(msg.Header.Get("Subject"))
	messageID := msg.Header.Get("Message-ID")
	if messageID == "" {
		messageID = uuid.NewString()
	}

	date, err := msg.Header.Date()
	if err != nil {
		date = time.Now()
	}

	toAddrs, _ := msg.Header.AddressList("To")
	var to []string
	for _, a := range toAddrs {
		if a.Address != "" {
			to = append(to, a.Address)
		} else {
			to = append(to, a.String())
		}
	}
	if len(to) == 0 && msg.Header.Get("To") != "" {
		to = []string{DecodeHeader(msg.Header.Get("To"))}
	}

	ccAddrs, _ := msg.Header.AddressList("Cc")
	var cc []string
	for _, a := range ccAddrs {
		if a.Address != "" {
			cc = append(cc, a.Address)
		} else {
			cc = append(cc, a.String())
		}
	}

	email := &ParsedEmail{
		MessageID: messageID,
		From:      from,
		To:        to,
		Cc:        cc,
		Subject:   subject,
		Date:      date,
		Headers:   headers,
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mediaType == "" {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			mr := multipart.NewReader(msg.Body, boundary)
			if err := parseMultipart(mr, email); err != nil {
				return nil, err
			}
		}
	} else {
		bodyBytes, err := readBodyWithEncoding(msg.Body, msg.Header.Get("Content-Transfer-Encoding"))
		if err == nil {
			content := string(bodyBytes)
			if strings.Contains(mediaType, "text/html") {
				email.HTMLBody = content
			} else {
				email.TextBody = content
			}
		}
	}

	return email, nil
}

func parseMultipart(mr *multipart.Reader, email *ParsedEmail) error {
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		partMediaType, partParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))

		filename := DecodeHeader(part.FileName())
		if filename == "" && dispParams != nil {
			filename = DecodeHeader(dispParams["filename"])
		}

		// Check if it's a nested multipart (e.g. multipart/alternative inside multipart/mixed)
		if strings.HasPrefix(partMediaType, "multipart/") {
			nestedBoundary := partParams["boundary"]
			if nestedBoundary != "" {
				nestedMR := multipart.NewReader(part, nestedBoundary)
				_ = parseMultipart(nestedMR, email)
			}
			continue
		}

		encoding := part.Header.Get("Content-Transfer-Encoding")
		bodyBytes, err := readBodyWithEncoding(part, encoding)
		if err != nil {
			continue
		}

		isAttachment := disposition == "attachment" || filename != ""
		if isAttachment {
			if filename == "" {
				ext := ".dat"
				if strings.Contains(partMediaType, "pdf") {
					ext = ".pdf"
				} else if strings.Contains(partMediaType, "image/png") {
					ext = ".png"
				} else if strings.Contains(partMediaType, "image/jpeg") {
					ext = ".jpg"
				}
				filename = "attachment" + ext
			}

			att, err := saveAttachment(filename, partMediaType, bodyBytes)
			if err == nil {
				email.Attachments = append(email.Attachments, att)
			}
		} else {
			content := string(bodyBytes)
			if strings.Contains(partMediaType, "text/html") {
				if email.HTMLBody == "" {
					email.HTMLBody = content
				} else {
					email.HTMLBody += "\n" + content
				}
			} else {
				if email.TextBody == "" {
					email.TextBody = content
				} else {
					email.TextBody += "\n" + content
				}
			}
		}
	}
	return nil
}

func readBodyWithEncoding(r io.Reader, encoding string) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "base64":
		b64Reader := base64.NewDecoder(base64.StdEncoding, r)
		return io.ReadAll(b64Reader)
	case "quoted-printable":
		qpReader := quotedprintable.NewReader(r)
		return io.ReadAll(qpReader)
	default:
		return io.ReadAll(r)
	}
}

// saveAttachment writes a file to data/uploads/:file_id/:filename and returns its EmailAttachment metadata.
func saveAttachment(rawFilename, contentType string, data []byte) (events.EmailAttachment, error) {
	fileID := uuid.NewString()
	filename := norm.NFC.String(filepath.Base(rawFilename))
	if filename == "" || filename == "." {
		filename = "attachment.dat"
	}

	uploadDir := filepath.Join("data", "uploads", fileID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return events.EmailAttachment{}, fmt.Errorf("create attachment directory: %w", err)
	}

	savePath := filepath.Join(uploadDir, filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return events.EmailAttachment{}, fmt.Errorf("write attachment file: %w", err)
	}

	return events.EmailAttachment{
		FileID:      fileID,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		URL:         fmt.Sprintf("/api/v1/uploads/%s/%s", fileID, filename),
	}, nil
}

// SaveBase64Attachment decodes a base64 string and saves it as an attachment.
func SaveBase64Attachment(filename, contentType, b64Data string) (events.EmailAttachment, error) {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return events.EmailAttachment{}, fmt.Errorf("decode base64: %w", err)
	}
	return saveAttachment(filename, contentType, data)
}

// ToMap converts a ParsedEmail into a generic blackboard payload map.
func (e *ParsedEmail) ToMap() map[string]interface{} {
	attachmentsList := make([]map[string]interface{}, 0, len(e.Attachments))
	for _, att := range e.Attachments {
		attachmentsList = append(attachmentsList, map[string]interface{}{
			"file_id":      att.FileID,
			"filename":     att.Filename,
			"content_type": att.ContentType,
			"size":         att.Size,
			"url":          att.URL,
		})
	}

	res := map[string]interface{}{
		"email_message_id":  e.MessageID,
		"email_from":        e.From,
		"email_to":          e.To,
		"email_cc":          e.Cc,
		"email_subject":     e.Subject,
		"email_date":        e.Date.Format(time.RFC3339),
		"email_text":        e.TextBody,
		"email_html":        e.HTMLBody,
		"email_attachments": attachmentsList,
		"has_attachments":   len(e.Attachments) > 0,
	}

	// If there is a primary attachment (like a single PDF invoice), expose it directly for simplicity
	if len(e.Attachments) > 0 {
		res["primary_attachment_url"] = e.Attachments[0].URL
		res["primary_attachment_filename"] = e.Attachments[0].Filename
	}

	return res
}
