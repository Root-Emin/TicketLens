package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
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

// Execute returns the thread in ascending order.
//
// The scope decides both which tickets are readable and whether internal notes
// come with them. A customer reading their own ticket gets the conversation
// without the notes; a customer naming somebody else's ticket gets not-found.
func (uc *ListMessagesUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID, scope Scope) ([]dto.MessageInfo, error) {
	// Confirms the ticket exists inside this organization before exposing its
	// messages; a foreign ticket reads as not-found.
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if err := scope.Guard(ticket); err != nil {
		return nil, err
	}

	messages, err := uc.messageRepo.ListByTicket(ctx, orgID, ticketID, scope.IncludesInternalNotes())
	if err != nil {
		return nil, err
	}
	return ToMessageInfos(messages), nil
}
