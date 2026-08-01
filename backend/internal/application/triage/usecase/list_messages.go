package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
)

// ListMessagesUseCase returns a ticket's conversation.
type ListMessagesUseCase struct {
	ticketRepo  repository.TicketRepository
	messageRepo repository.TicketMessageRepository
}

// NewListMessagesUseCase creates a new ListMessagesUseCase.
func NewListMessagesUseCase(
	ticketRepo repository.TicketRepository,
	messageRepo repository.TicketMessageRepository,
) *ListMessagesUseCase {
	return &ListMessagesUseCase{ticketRepo: ticketRepo, messageRepo: messageRepo}
}

// Execute returns the thread in ascending order. Customer-facing callers pass
// includeInternal=false and never receive internal notes.
func (uc *ListMessagesUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID, includeInternal bool) ([]dto.MessageInfo, error) {
	// Confirms the ticket exists inside this organization before exposing its
	// messages; a foreign ticket reads as not-found.
	if _, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID); err != nil {
		return nil, err
	}

	messages, err := uc.messageRepo.ListByTicket(ctx, orgID, ticketID, includeInternal)
	if err != nil {
		return nil, err
	}
	return ToMessageInfos(messages), nil
}
