package email

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// IMAPConfig holds connection parameters for an IMAP mailbox.
type IMAPConfig struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`     // default: 993
	Username string        `json:"username"`
	Password string        `json:"password"`
	Mailbox  string        `json:"mailbox"`  // default: "INBOX"
	UseTLS   bool          `json:"use_tls"`  // default: true
	Timeout  time.Duration `json:"timeout"`  // default: 30s
}

// IMAPClient provides basic RFC 3501 operations for retrieving unseen emails.
type IMAPClient struct {
	conn   net.Conn
	reader *bufio.Reader
	seq    int
}

// DialIMAP connects to an IMAP server using TLS or plain TCP.
func DialIMAP(cfg IMAPConfig) (*IMAPClient, error) {
	if cfg.Port == 0 {
		if cfg.UseTLS {
			cfg.Port = 993
		} else {
			cfg.Port = 143
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var conn net.Conn
	var err error

	if cfg.UseTLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}, "tcp", addr, &tls.Config{
			ServerName: cfg.Host,
		})
	} else {
		conn, err = net.DialTimeout("tcp", addr, cfg.Timeout)
	}

	if err != nil {
		return nil, fmt.Errorf("dial imap server %s: %w", addr, err)
	}

	client := &IMAPClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}

	// Read initial greeting
	greeting, err := client.readLine()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("read greeting: %w", err)
	}
	if !strings.HasPrefix(greeting, "* OK") && !strings.HasPrefix(greeting, "* PREAUTH") {
		_ = client.Close()
		return nil, fmt.Errorf("unexpected imap greeting: %s", greeting)
	}

	return client, nil
}

func (c *IMAPClient) nextTag() string {
	c.seq++
	return fmt.Sprintf("A%04d", c.seq)
}

func (c *IMAPClient) readLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// execCommand sends an IMAP command tagged with a unique tag and waits for its tagged completion.
func (c *IMAPClient) execCommand(cmd string) ([]string, error) {
	tag := c.nextTag()
	fullCmd := fmt.Sprintf("%s %s\r\n", tag, cmd)
	if _, err := c.conn.Write([]byte(fullCmd)); err != nil {
		return nil, fmt.Errorf("write command: %w", err)
	}

	var untagged []string
	for {
		line, err := c.readLine()
		if err != nil {
			return untagged, fmt.Errorf("read response: %w", err)
		}

		if strings.HasPrefix(line, tag+" ") {
			status := strings.TrimPrefix(line, tag+" ")
			if strings.HasPrefix(strings.ToUpper(status), "OK") {
				return untagged, nil
			}
			return untagged, fmt.Errorf("imap error: %s", status)
		}

		untagged = append(untagged, line)
	}
}

// Login authenticates with username and password.
func (c *IMAPClient) Login(username, password string) error {
	// Escape backslashes and double quotes in credentials
	escapedUser := strings.ReplaceAll(strings.ReplaceAll(username, `\`, `\\`), `"`, `\"`)
	escapedPass := strings.ReplaceAll(strings.ReplaceAll(password, `\`, `\\`), `"`, `\"`)
	_, err := c.execCommand(fmt.Sprintf(`LOGIN "%s" "%s"`, escapedUser, escapedPass))
	return err
}

// SelectMailbox opens a folder (e.g. "INBOX").
func (c *IMAPClient) SelectMailbox(mailbox string) error {
	if mailbox == "" {
		mailbox = "INBOX"
	}
	_, err := c.execCommand(fmt.Sprintf(`SELECT "%s"`, mailbox))
	return err
}

// SearchUnseen finds sequence numbers of unread messages.
func (c *IMAPClient) SearchUnseen() ([]int, error) {
	lines, err := c.execCommand("SEARCH UNSEEN")
	if err != nil {
		return nil, err
	}

	var seqs []int
	for _, l := range lines {
		if strings.HasPrefix(l, "* SEARCH") {
			parts := strings.Fields(strings.TrimPrefix(l, "* SEARCH"))
			for _, p := range parts {
				if num, err := strconv.Atoi(p); err == nil {
					seqs = append(seqs, num)
				}
			}
		}
	}
	return seqs, nil
}

// FetchRawMessage fetches the complete RFC822 literal bytes for a sequence number.
func (c *IMAPClient) FetchRawMessage(seq int) ([]byte, error) {
	tag := c.nextTag()
	cmd := fmt.Sprintf("%s FETCH %d (BODY[] FLAGS)\r\n", tag, seq)
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return nil, err
	}

	var rawBytes []byte
	var taggedStatus string

	for {
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(line, tag+" ") {
			taggedStatus = strings.TrimPrefix(line, tag+" ")
			break
		}

		// Look for literal length header e.g.: * 1 FETCH (BODY[] {1234}
		if idx := strings.Index(line, "{"); idx != -1 && strings.HasSuffix(line, "}") {
			lenStr := line[idx+1 : len(line)-1]
			if byteLen, err := strconv.Atoi(lenStr); err == nil && byteLen > 0 {
				literalBuf := make([]byte, byteLen)
				if _, err := io.ReadFull(c.reader, literalBuf); err != nil {
					return nil, fmt.Errorf("read literal data: %w", err)
				}
				rawBytes = literalBuf
			}
		}
	}

	if !strings.HasPrefix(strings.ToUpper(taggedStatus), "OK") {
		return nil, fmt.Errorf("fetch failed: %s", taggedStatus)
	}

	if len(rawBytes) == 0 {
		return nil, fmt.Errorf("no message body returned for seq %d", seq)
	}

	return rawBytes, nil
}

// MarkSeen adds the \Seen flag to a sequence number.
func (c *IMAPClient) MarkSeen(seq int) error {
	_, err := c.execCommand(fmt.Sprintf("STORE %d +FLAGS (\\Seen)", seq))
	return err
}

// Close gracefully logs out and closes the TCP connection.
func (c *IMAPClient) Close() error {
	_, _ = c.execCommand("LOGOUT")
	return c.conn.Close()
}
