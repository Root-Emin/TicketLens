package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	triageEvent "github.com/Root-Emin/TicketLens/internal/domain/triage/event"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/google/uuid"
)

// CreateTicketUseCase raises a ticket together with its first message.
type CreateTicketUseCase struct {
	ticketRepo     repository.TicketRepository
	messageRepo    repository.TicketMessageRepository
	customerRepo   repository.CustomerRepository
	departmentRepo repository.DepartmentRepository
	txManager      repository.TxManager
	eventBus       events.EventBus
	assembler      *ticketAssembler
}

// NewCreateTicketUseCase creates a new CreateTicketUseCase.
func NewCreateTicketUseCase(
	ticketRepo repository.TicketRepository,
	messageRepo repository.TicketMessageRepository,
	customerRepo repository.CustomerRepository,
	departmentRepo repository.DepartmentRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
	txManager repository.TxManager,
	eventBus events.EventBus,
) *CreateTicketUseCase {
	return &CreateTicketUseCase{
		ticketRepo:     ticketRepo,
		messageRepo:    messageRepo,
		customerRepo:   customerRepo,
		departmentRepo: departmentRepo,
		txManager:      txManager,
		eventBus:       eventBus,
		assembler:      newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute creates the ticket, stores the body as its first message, and
// announces it so the classification consumer can pick it up.
//
// Priority and department are not taken from the request: the ticket starts in
// the default department at normal priority and stays there until the first
// analysis arrives.
//
// The ticket and its first message are written inside one transaction, so a
// failure on the second write can no longer leave a ticket with no description.
func (uc *CreateTicketUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.CreateTicketRequest, scope Scope) (*dto.TicketDetail, error) {
	customerID, err := uc.ownerFor(req, scope)
	if err != nil {
		return nil, err
	}

	// Resolving through the org-scoped repository is what stops an agent from
	// attaching a ticket to another tenant's customer: this returns not-found.
	customer, err := uc.customerRepo.GetByID(ctx, orgID, customerID)
	if err != nil {
		return nil, err
	}

	department, err := uc.departmentRepo.GetDefault(ctx, orgID)
	if err != nil {
		return nil, err
	}

	ticket := &model.Ticket{
		OrganizationID: orgID,
		CustomerID:     customer.ID,
		DepartmentID:   department.ID,
		Subject:        strings.TrimSpace(req.Subject),
		Status:         model.TicketStatusOpen,
		Priority:       model.TicketPriorityNormal,
	}

	// The first message is the ticket's description; there is no description
	// column on tickets.
	authorID := customer.ID
	message := &model.TicketMessage{
		OrganizationID: orgID,
		TicketID:       ticket.ID,
		AuthorType:     model.AuthorTypeCustomer,
		AuthorID:       &authorID,
		Body:           req.Body,
		IsInternal:     false,
	}

	if err := uc.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		if err := uc.ticketRepo.Create(txCtx, ticket); err != nil {
			return err
		}
		// message.TicketID is set here rather than above because ticketRepo.Create
		// assigns the id; the two writes share one transaction so this ordering is
		// safe.
		message.TicketID = ticket.ID
		return uc.messageRepo.Create(txCtx, message)
	}); err != nil {
		return nil, err
	}

	_ = uc.eventBus.Publish(ctx, events.TopicTriage, triageEvent.TicketCreated{
		TicketID:       ticket.ID,
		OrganizationID: orgID,
		CustomerID:     customer.ID,
		DepartmentID:   department.ID,
		Subject:        ticket.Subject,
		Body:           message.Body,
		Timestamp:      time.Now().UTC(),
	})

	return uc.assembler.toDetail(ctx, orgID, ticket, scope.IncludesInternalNotes())
}

// ownerFor decides whose ticket this is.
//
// A customer's own id always wins and the body's customer_id is discarded, not
// validated against theirs: a request that names somebody else is not an error
// to report back, it is a field that was never the client's to set. Staff, who
// legitimately raise tickets on a customer's behalf, must still name one.
func (uc *CreateTicketUseCase) ownerFor(req dto.CreateTicketRequest, scope Scope) (uuid.UUID, error) {
	if scope.IsCustomer() {
		owner := scope.OwnerID()
		if owner == uuid.Nil {
			return uuid.Nil, domainErr.New(domainErr.ErrForbidden,
				"this account is not linked to a customer record", nil)
		}
		return owner, nil
	}

	if req.CustomerID == uuid.Nil {
		return uuid.Nil, domainErr.New(domainErr.ErrValidation,
			"customer_id is required when raising a ticket on a customer's behalf", nil)
	}
	return req.CustomerID, nil
}
