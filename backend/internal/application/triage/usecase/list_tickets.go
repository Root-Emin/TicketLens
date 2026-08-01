package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
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
// TODO(customer-portal): a caller holding only ticket:read_own must have
// filter.CustomerID pinned to their own customer id. The filter already
// supports it; the customer token carrying that id does not exist yet.
func (uc *ListTicketsUseCase) Execute(ctx context.Context, orgID uuid.UUID, filter repository.TicketFilter, page dto.PageParams) (dto.ListResponse[dto.TicketListItem], error) {
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
