package email

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// EmailHandler is invoked whenever a new email is ingested by the poller.
type EmailHandler func(ctx context.Context, email *ParsedEmail) error

// IMAPPoller periodically polls an IMAP mailbox for unread messages.
type IMAPPoller struct {
	config   IMAPConfig
	interval time.Duration
	handler  EmailHandler
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewIMAPPoller constructs a background IMAP poller.
func NewIMAPPoller(cfg IMAPConfig, interval time.Duration, handler EmailHandler) *IMAPPoller {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &IMAPPoller{
		config:   cfg,
		interval: interval,
		handler:  handler,
	}
}

// Start begins the polling loop in a background goroutine.
func (p *IMAPPoller) Start(parentCtx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("imap poller already running")
	}

	ctx, cancel := context.WithCancel(parentCtx)
	p.cancel = cancel
	p.running = true
	p.wg.Add(1)

	go p.loop(ctx)
	slog.Info("Started background IMAP poller", "host", p.config.Host, "mailbox", p.config.Mailbox, "interval", p.interval)
	return nil
}

// Stop terminates the polling loop and waits for any ongoing poll to complete.
func (p *IMAPPoller) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.cancel()
	p.running = false
	p.mu.Unlock()

	p.wg.Wait()
	slog.Info("Stopped background IMAP poller", "host", p.config.Host)
}

func (p *IMAPPoller) loop(ctx context.Context) {
	defer p.wg.Done()

	// Perform initial poll immediately
	p.pollOnce(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollOnce(ctx)
		}
	}
}

// PollOnce connects to the IMAP server, retrieves all UNSEEN emails, processes them, and marks them seen.
func (p *IMAPPoller) PollOnce(ctx context.Context) error {
	return p.pollOnce(ctx)
}

func (p *IMAPPoller) pollOnce(ctx context.Context) error {
	client, err := DialIMAP(p.config)
	if err != nil {
		slog.Error("IMAP connection failed", "host", p.config.Host, "error", err)
		return err
	}
	defer client.Close()

	if err := client.Login(p.config.Username, p.config.Password); err != nil {
		slog.Error("IMAP login failed", "user", p.config.Username, "error", err)
		return err
	}

	if err := client.SelectMailbox(p.config.Mailbox); err != nil {
		slog.Error("IMAP select mailbox failed", "mailbox", p.config.Mailbox, "error", err)
		return err
	}

	seqs, err := client.SearchUnseen()
	if err != nil {
		slog.Error("IMAP search unseen failed", "error", err)
		return err
	}

	if len(seqs) == 0 {
		return nil
	}

	slog.Info("Found unseen emails in mailbox", "count", len(seqs), "mailbox", p.config.Mailbox)

	for _, seq := range seqs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rawBytes, err := client.FetchRawMessage(seq)
		if err != nil {
			slog.Error("Failed to fetch message", "seq", seq, "error", err)
			continue
		}

		parsed, err := ParseRFC822(bytes.NewReader(rawBytes))
		if err != nil {
			slog.Error("Failed to parse message RFC822", "seq", seq, "error", err)
			continue
		}

		if p.handler != nil {
			if err := p.handler(ctx, parsed); err != nil {
				slog.Error("Email handler failed", "subject", parsed.Subject, "from", parsed.From, "error", err)
				// Do not mark seen so it can be retried or inspected
				continue
			}
		}

		// Mark seen once handler succeeds
		if err := client.MarkSeen(seq); err != nil {
			slog.Warn("Failed to mark email seen", "seq", seq, "error", err)
		}
	}

	return nil
}
