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
// Authorship is derived from the scope, never from the request body: a customer
// writes as themselves on their own ticket, and everybody else writes as an
// agent. A client that could name its own author_type could impersonate support
// inside the customer's own thread.
//
// is_internal is forced false for a customer for the same reason. The field is
// honest on a staff request and meaningless on a customer one — a note nobody
// but the author can read is not a note, and accepting the flag would let a
// customer write into the private half of the thread.
func (uc *CreateMessageUseCase) Execute(
	ctx context.Context,
	orgID, ticketID, actorID uuid.UUID,
	req dto.CreateMessageRequest,
	scope Scope,
) (*dto.MessageInfo, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if err := scope.Guard(ticket); err != nil {
		return nil, err
	}

	authorType := model.AuthorTypeAgent
	authorID := actorID
	isInternal := req.IsInternal
	if scope.IsCustomer() {
		authorType = model.AuthorTypeCustomer
		authorID = scope.OwnerID()
		isInternal = false
	}

	message := &model.TicketMessage{
		OrganizationID: orgID,
		TicketID:       ticketID,
		AuthorType:     authorType,
		Body:           req.Body,
		IsInternal:     isInternal,
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
