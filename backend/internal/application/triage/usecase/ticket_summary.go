package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// TicketSummaryUseCase serves the portal dashboard's counters.
//
// It exists so a customer does not have to download a page of tickets and count
// them in the browser, which is what the portal did while this endpoint was
// missing: correct only up to the page size, and wasteful at any size.
type TicketSummaryUseCase struct {
	statsRepo repository.StatsRepository
}

// NewTicketSummaryUseCase creates a new TicketSummaryUseCase.
func NewTicketSummaryUseCase(statsRepo repository.StatsRepository) *TicketSummaryUseCase {
	return &TicketSummaryUseCase{statsRepo: statsRepo}
}

// Execute aggregates the caller's tickets.
//
// The scope decides the denominator: a customer counts their own requests, an
// agent counts the organization's. Nothing here reads a customer id from the
// request, so there is no version of this call that reports on someone else.
func (uc *TicketSummaryUseCase) Execute(ctx context.Context, orgID uuid.UUID, scope Scope) (*dto.TicketSummaryInfo, error) {
	summary, err := uc.statsRepo.Summary(ctx, orgID, scope.CustomerFilter())
	if err != nil {
		return nil, err
	}

	byStatus := make(map[string]int, len(summary.ByStatus))
	for status, count := range summary.ByStatus {
		byStatus[string(status)] = count
	}

	return &dto.TicketSummaryInfo{
		Total:                   summary.Total,
		Open:                    summary.Open(),
		WaitingCustomer:         summary.WaitingCustomer(),
		Resolved:                summary.Resolved(),
		ByStatus:                byStatus,
		AvgFirstResponseMinutes: summary.AvgFirstResponse,
		AvgResolutionMinutes:    summary.AvgResolution,
	}, nil
}
