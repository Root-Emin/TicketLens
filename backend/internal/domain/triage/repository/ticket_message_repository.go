package repository

import (
	"context"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// TicketMessageRepository defines the interface for ticket message persistence.
type TicketMessageRepository interface {
	Create(ctx context.Context, message *model.TicketMessage) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.TicketMessage, error)
	// ListByTicket returns a ticket's thread in ascending order. When
	// includeInternal is false, internal notes are omitted — that is the shape
	// the customer portal receives.
	ListByTicket(ctx context.Context, orgID, ticketID uuid.UUID, includeInternal bool) ([]*model.TicketMessage, error)
	// CountByTickets returns message counts keyed by ticket ID so the queue
	// listing can render message_count without an N+1.
	CountByTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]int, error)

	// PreviewByTickets returns each ticket's opening message, truncated to
	// maxRunes, keyed by ticket ID. Tickets are described by their first
	// message, so a list that shows a preview needs this; doing it per row is
	// the N+1 the other batch methods here exist to avoid.
	PreviewByTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID, maxRunes int) (map[uuid.UUID]string, error)

	// FirstResponseByTickets returns when support first replied to each ticket,
	// keyed by ticket ID. Internal notes and the customer's own messages do not
	// count as a response — the metric is what the requester waited for, not
	// what the queue did internally.
	FirstResponseByTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]time.Time, error)
}
