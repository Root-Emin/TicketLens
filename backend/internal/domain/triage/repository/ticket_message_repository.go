package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
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
}
