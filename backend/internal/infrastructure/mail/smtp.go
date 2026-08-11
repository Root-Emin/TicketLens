package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/port"
)

// SMTPConfig is what SMTPMailer needs to reach a relay.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// Timeout bounds the whole conversation with the relay: dial, handshake and
	// delivery. Without it a relay that accepts the connection and then stops
	// answering holds the calling request open indefinitely.
	Timeout time.Duration
}

// SMTPMailer delivers mail through an SMTP relay.
//
// Built on net/smtp from the standard library rather than a mail package: the
// messages are plain text with no attachments and no alternative parts, which is
// the one case net/smtp covers completely. A dependency would earn its place
// when branded HTML mail arrives, not before.
type SMTPMailer struct {
	cfg SMTPConfig
}

// Verify interface compliance at compile time.
var _ port.Mailer = (*SMTPMailer)(nil)

// NewSMTPMailer creates a mailer that sends through the configured relay.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, msg port.Message) error {
	// Header injection: a newline in either field would let the caller append
	// headers of its own — a second Bcc, a different From. Callers are internal
	// today, but the values reach here from a request body (the invited address)
	// and the check belongs at the point where the header is built.
	if strings.ContainsAny(msg.To, "\r\n") || strings.ContainsAny(msg.Subject, "\r\n") {
		return fmt.Errorf("mail: header field contains a line break")
	}

	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))

	dialer := &net.Dialer{Timeout: m.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", addr, err)
	}
	// Closed explicitly on the success path by Quit; this covers every early
	// return below, where the connection would otherwise leak.
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(m.cfg.Timeout))
	}

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("mail: smtp handshake: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS whenever the relay offers it. Not optional in practice: the
	// credentials below are sent on this connection, and PlainAuth itself
	// refuses to run unencrypted.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}

	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("mail: from %s: %w", m.cfg.From, err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("mail: rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := w.Write(m.buildMessage(msg)); err != nil {
		return fmt.Errorf("mail: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close body: %w", err)
	}

	return client.Quit()
}

// buildMessage renders the RFC 5322 message. CRLF line endings throughout, as
// the format requires — a bare newline is accepted by some relays and mangles
// the body on others.
func (m *SMTPMailer) buildMessage(msg port.Message) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(msg.Body, "\n", "\r\n"))
	return []byte(b.String())
}
