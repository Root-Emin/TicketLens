// Package mail provides the outbound adapters for port.Mailer.
package mail

import (
	"context"
	"log/slog"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/port"
)

// LogMailer writes messages to the log instead of delivering them.
//
// This is the default when no SMTP host is configured, which makes local
// development and the test suite work with no mail server: the invitation link
// is read out of the server log. It mirrors how the classifier falls back to its
// keyword stub when CLASSIFIER_URL is empty — the feature stays exercisable, and
// the absence is visible rather than silent.
//
// The body is logged in full. That is correct here and would not be for a real
// mailer: these messages carry single-use tokens, so anyone who can read the log
// can accept the invitation. A deployment that configures SMTP never constructs
// this type; a deployment that does not is by definition not handling real
// recipients.
type LogMailer struct {
	log *slog.Logger
}

// Verify interface compliance at compile time.
var _ port.Mailer = (*LogMailer)(nil)

// NewLogMailer creates a mailer that logs rather than sends.
func NewLogMailer(log *slog.Logger) *LogMailer {
	return &LogMailer{log: log}
}

func (m *LogMailer) Send(_ context.Context, msg port.Message) error {
	m.log.Info("email not sent: no SMTP configured, logging instead",
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.Body,
	)
	return nil
}
