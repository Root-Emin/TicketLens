package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// ListTicketsUseCase serves the ticket queue.
type ListTicketsUseCase struct {
	ticketRepo repository.TicketRepository
	assembler  *ticketAssembler
}

// NewListTicketsUseCase creates a new ListTicketsUseCase.
func NewListTicketsUseCase(
	ticketRepo repository.TicketRepository,
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.TicketMessageRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
) *ListTicketsUseCase {
	return &ListTicketsUseCase{
		ticketRepo: ticketRepo,
		assembler:  newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute returns a filtered, sorted page of tickets.
//
// A customer scope overwrites filter.CustomerID rather than defaulting it. The
// value arrives from the request's query string, and honouring it here would
// let anyone read another customer's queue by naming them — the whole point is
// that this field is the server's to set.
func (uc *ListTicketsUseCase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	filter repository.TicketFilter,
	page dto.PageParams,
	scope Scope,
) (dto.ListResponse[dto.TicketListItem], error) {
	if scope.IsCustomer() {
		filter.CustomerID = scope.CustomerFilter()
	}

	tickets, total, err := uc.ticketRepo.ListByOrg(ctx, orgID, filter, page.Offset(), page.Limit())
	if err != nil {
		return dto.ListResponse[dto.TicketListItem]{}, err
	}

	items, err := uc.assembler.toListItems(ctx, orgID, tickets)
	if err != nil {
		return dto.ListResponse[dto.TicketListItem]{}, err
	}

	return dto.NewListResponse(items, page, total), nil
}
