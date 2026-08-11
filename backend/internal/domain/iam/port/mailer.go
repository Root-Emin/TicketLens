// Package port holds the outbound interfaces the IAM context depends on.
// Nothing here may import an infrastructure package: adapters live under
// internal/infrastructure and depend on these definitions, never the reverse.
package port

import "context"

// Message is one outgoing email.
//
// Plain text only, deliberately. Every message this context sends is a short
// transactional notice built around a single link, and an HTML alternative would
// add templating and a second body to keep in agreement for no gain the reader
// can see. When branded mail arrives it should extend this type rather than have
// callers assemble MIME themselves.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer delivers a message.
//
// Implementations must be safe for concurrent use. Callers should treat a
// failure as a degraded outcome rather than a failed operation wherever the
// message is not the only way to complete the task: an invitation whose email
// bounced is still a valid invitation, and failing the request would destroy a
// record the administrator can still act on by copying the link.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
