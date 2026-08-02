package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// AIAnalysisRepository defines the interface for classifier output persistence.
// The table is append-only: there is no Update.
type AIAnalysisRepository interface {
	Create(ctx context.Context, analysis *model.AIAnalysis) error
	// ListByTicket returns a ticket's full analysis history, newest first.
	ListByTicket(ctx context.Context, orgID, ticketID uuid.UUID) ([]*model.AIAnalysis, error)
	// GetLatestByTicket returns the most recent analysis for a ticket, or
	// ErrNotFound while classification is still pending.
	GetLatestByTicket(ctx context.Context, orgID, ticketID uuid.UUID) (*model.AIAnalysis, error)
	// LatestForTickets returns the most recent analysis per ticket, keyed by
	// ticket ID. Tickets without an analysis are absent from the map, which is
	// how the list view renders a null latest_analysis without an N+1.
	LatestForTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]*model.AIAnalysis, error)
}
