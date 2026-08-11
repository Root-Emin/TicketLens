package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// GetTicketUseCase returns one ticket with its thread and analysis history.
type GetTicketUseCase struct {
	ticketRepo repository.TicketRepository
	assembler  *ticketAssembler
}

// NewGetTicketUseCase creates a new GetTicketUseCase.
func NewGetTicketUseCase(
	ticketRepo repository.TicketRepository,
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.TicketMessageRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
) *GetTicketUseCase {
	return &GetTicketUseCase{
		ticketRepo: ticketRepo,
		assembler:  newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute fetches a ticket.
//
// Two boundaries are enforced here and both answer not-found rather than
// forbidden, because the API must not confirm that a row it will not show you
// exists:
//
//   - the repository is organization-scoped, so another tenant's ticket is
//     already invisible;
//   - the scope guard rejects a ticket that belongs to a different customer.
//
// Scoping the list without scoping this would leave the queue private and the
// detail wide open — a customer could still read any ticket by putting its id
// in the URL.
//
// Whether internal notes are included comes from the scope too, so a customer
// cannot receive one through a caller that forgot to pass the flag.
func (uc *GetTicketUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID, scope Scope) (*dto.TicketDetail, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if err := scope.Guard(ticket); err != nil {
		return nil, err
	}
	return uc.assembler.toDetail(ctx, orgID, ticket, scope.IncludesInternalNotes())
}
