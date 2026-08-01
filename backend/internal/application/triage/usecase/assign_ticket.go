package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	triageEvent "github.com/masterfabric-go/masterfabric/internal/domain/triage/event"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/events"
)

// AssignTicketUseCase sets or clears a ticket's assignee.
type AssignTicketUseCase struct {
	ticketRepo repository.TicketRepository
	roleRepo   iamRepo.RoleRepository
	eventBus   events.EventBus
	assembler  *ticketAssembler
}

// NewAssignTicketUseCase creates a new AssignTicketUseCase.
func NewAssignTicketUseCase(
	ticketRepo repository.TicketRepository,
	roleRepo iamRepo.RoleRepository,
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.TicketMessageRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
	eventBus events.EventBus,
) *AssignTicketUseCase {
	return &AssignTicketUseCase{
		ticketRepo: ticketRepo,
		roleRepo:   roleRepo,
		eventBus:   eventBus,
		assembler:  newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute assigns the ticket, or unassigns it when assigneeID is nil.
// Assigning somebody who holds no role in this organization is a 422.
func (uc *AssignTicketUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID, req dto.AssignTicketRequest) (*dto.TicketDetail, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}

	if req.AssigneeID == nil || *req.AssigneeID == uuid.Nil {
		ticket.AssigneeID = nil
	} else {
		// Membership is expressed through role assignments, so having any role
		// in this organization is what makes a user assignable.
		roles, err := uc.roleRepo.GetUserRoles(ctx, *req.AssigneeID, orgID)
		if err != nil {
			return nil, err
		}
		if len(roles) == 0 {
			return nil, domainErr.New(domainErr.ErrValidation,
				"assignee is not a member of this organization", nil)
		}
		ticket.AssigneeID = req.AssigneeID
	}

	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, err
	}

	_ = uc.eventBus.Publish(ctx, events.TopicTriage, triageEvent.TicketAssigned{
		TicketID:       ticket.ID,
		OrganizationID: orgID,
		AssigneeID:     ticket.AssigneeID,
		Timestamp:      time.Now().UTC(),
	})

	return uc.assembler.toDetail(ctx, orgID, ticket, true)
}
