package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// CreateMessageUseCase appends a message to a ticket.
type CreateMessageUseCase struct {
	ticketRepo  repository.TicketRepository
	messageRepo repository.TicketMessageRepository
}

// NewCreateMessageUseCase creates a new CreateMessageUseCase.
func NewCreateMessageUseCase(
	ticketRepo repository.TicketRepository,
	messageRepo repository.TicketMessageRepository,
) *CreateMessageUseCase {
	return &CreateMessageUseCase{ticketRepo: ticketRepo, messageRepo: messageRepo}
}

// Execute appends a message.
//
// authorType and authorID are supplied by the caller from the authenticated
// token — never from the request body — so a client cannot post as somebody
// else.
func (uc *CreateMessageUseCase) Execute(
	ctx context.Context,
	orgID, ticketID uuid.UUID,
	authorType model.AuthorType,
	authorID uuid.UUID,
	req dto.CreateMessageRequest,
) (*dto.MessageInfo, error) {
	if _, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID); err != nil {
		return nil, err
	}

	message := &model.TicketMessage{
		OrganizationID: orgID,
		TicketID:       ticketID,
		AuthorType:     authorType,
		Body:           req.Body,
		IsInternal:     req.IsInternal,
	}
	if authorID != uuid.Nil {
		id := authorID
		message.AuthorID = &id
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	info := ToMessageInfos([]*model.TicketMessage{message})[0]
	return &info, nil
}
