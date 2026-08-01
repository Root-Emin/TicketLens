package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	triageEvent "github.com/masterfabric-go/masterfabric/internal/domain/triage/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

// CreateTicketUseCase raises a ticket together with its first message.
type CreateTicketUseCase struct {
	ticketRepo     repository.TicketRepository
	messageRepo    repository.TicketMessageRepository
	customerRepo   repository.CustomerRepository
	departmentRepo repository.DepartmentRepository
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
	eventBus events.EventBus,
) *CreateTicketUseCase {
	return &CreateTicketUseCase{
		ticketRepo:     ticketRepo,
		messageRepo:    messageRepo,
		customerRepo:   customerRepo,
		departmentRepo: departmentRepo,
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
// NOTE: the ticket and its first message are two statements without an
// enclosing transaction — the repository interfaces expose no transaction
// handle. A failure between them leaves a ticket with no description.
func (uc *CreateTicketUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.CreateTicketRequest) (*dto.TicketDetail, error) {
	// Resolving through the org-scoped repository is what stops an agent from
	// attaching a ticket to another tenant's customer: this returns not-found.
	customer, err := uc.customerRepo.GetByID(ctx, orgID, req.CustomerID)
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
	if err := uc.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, err
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
	if err := uc.messageRepo.Create(ctx, message); err != nil {
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

	return uc.assembler.toDetail(ctx, orgID, ticket, true)
}
