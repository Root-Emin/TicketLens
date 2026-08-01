package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
)

// ListAnalysesUseCase returns a ticket's full classifier history — the "lens"
// view: what the model predicted, how sure it was, and whether a human
// disagreed.
type ListAnalysesUseCase struct {
	ticketRepo   repository.TicketRepository
	analysisRepo repository.AIAnalysisRepository
}

// NewListAnalysesUseCase creates a new ListAnalysesUseCase.
func NewListAnalysesUseCase(
	ticketRepo repository.TicketRepository,
	analysisRepo repository.AIAnalysisRepository,
) *ListAnalysesUseCase {
	return &ListAnalysesUseCase{ticketRepo: ticketRepo, analysisRepo: analysisRepo}
}

// Execute returns every analysis of a ticket, newest first.
func (uc *ListAnalysesUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID) ([]dto.AnalysisInfo, error) {
	if _, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID); err != nil {
		return nil, err
	}

	analyses, err := uc.analysisRepo.ListByTicket(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	return ToAnalysisInfos(analyses), nil
}
