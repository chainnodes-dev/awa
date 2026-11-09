package email

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRFC822_SimpleText(t *testing.T) {
	raw := "From: alice@example.com\r\n" +
		"To: bob@example.com\r\n" +
		"Subject: Hello World\r\n" +
		"Date: Mon, 9 Nov 2026 12:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"This is a test email message body."

	parsed, err := ParseRFC822(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseRFC822 failed: %v", err)
	}

	if parsed.From != "alice@example.com" {
		t.Errorf("expected from alice@example.com, got %s", parsed.From)
	}
	if len(parsed.To) != 1 || parsed.To[0] != "bob@example.com" {
		t.Errorf("unexpected to: %v", parsed.To)
	}
	if parsed.Subject != "Hello World" {
		t.Errorf("unexpected subject: %s", parsed.Subject)
	}
	if !strings.Contains(parsed.TextBody, "This is a test email message body.") {
		t.Errorf("unexpected text body: %s", parsed.TextBody)
	}
}

func TestParseRFC822_MultipartWithAttachment(t *testing.T) {
	pdfDummy := "%PDF-1.4 dummy invoice content"
	b64PDF := base64.StdEncoding.EncodeToString([]byte(pdfDummy))

	raw := "From: vendor@billing.com\r\n" +
		"To: accounts@myorg.com\r\n" +
		"Subject: Invoice #1042\r\n" +
		"Date: Mon, 9 Nov 2026 14:30:00 +0000\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=\"BOUNDARY_123\"\r\n" +
		"\r\n" +
		"--BOUNDARY_123\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"Please find attached your invoice for November.\r\n" +
		"--BOUNDARY_123\r\n" +
		"Content-Type: application/pdf; name=\"invoice_1042.pdf\"\r\n" +
		"Content-Disposition: attachment; filename=\"invoice_1042.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		b64PDF + "\r\n" +
		"--BOUNDARY_123--\r\n"

	parsed, err := ParseRFC822(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseRFC822 failed: %v", err)
	}

	if parsed.Subject != "Invoice #1042" {
		t.Errorf("unexpected subject: %s", parsed.Subject)
	}
	if !strings.Contains(parsed.TextBody, "Please find attached your invoice") {
		t.Errorf("unexpected text body: %s", parsed.TextBody)
	}

	if len(parsed.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(parsed.Attachments))
	}

	att := parsed.Attachments[0]
	if att.Filename != "invoice_1042.pdf" {
		t.Errorf("expected invoice_1042.pdf, got %s", att.Filename)
	}
	if att.Size != int64(len(pdfDummy)) {
		t.Errorf("expected size %d, got %d", len(pdfDummy), att.Size)
	}

	defer func() {
		_ = os.RemoveAll(filepath.Join("data", "uploads", att.FileID))
	}()

	// Verify attachment was saved to disk
	savedPath := filepath.Join("data", "uploads", att.FileID, att.Filename)
	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("failed to read saved attachment: %v", err)
	}
	if string(data) != pdfDummy {
		t.Errorf("content mismatch in saved file")
	}

	// Verify ToMap conversion
	m := parsed.ToMap()
	if m["email_subject"] != "Invoice #1042" {
		t.Errorf("ToMap email_subject mismatch")
	}
	if m["primary_attachment_filename"] != "invoice_1042.pdf" {
		t.Errorf("ToMap primary_attachment_filename mismatch")
	}
}

func TestSaveBase64Attachment(t *testing.T) {
	content := "Sample file bytes"
	b64 := base64.StdEncoding.EncodeToString([]byte(content))

	att, err := SaveBase64Attachment("sample.txt", "text/plain", b64)
	if err != nil {
		t.Fatalf("SaveBase64Attachment failed: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(filepath.Join("data", "uploads", att.FileID))
	}()

	if att.Filename != "sample.txt" || att.Size != int64(len(content)) {
		t.Errorf("unexpected attachment metadata: %+v", att)
	}
}
