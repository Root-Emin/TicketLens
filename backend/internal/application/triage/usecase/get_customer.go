package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// recentTicketLimit is how many of a customer's latest tickets the detail view
// carries, per the API contract.
const recentTicketLimit = 5

// GetCustomerUseCase returns one customer with their ticket history summary.
type GetCustomerUseCase struct {
	customerRepo repository.CustomerRepository
	ticketRepo   repository.TicketRepository
	assembler    *ticketAssembler
}

// NewGetCustomerUseCase creates a new GetCustomerUseCase.
func NewGetCustomerUseCase(
	customerRepo repository.CustomerRepository,
	ticketRepo repository.TicketRepository,
	departmentRepo repository.DepartmentRepository,
	messageRepo repository.TicketMessageRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
) *GetCustomerUseCase {
	return &GetCustomerUseCase{
		customerRepo: customerRepo,
		ticketRepo:   ticketRepo,
		assembler:    newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute returns the customer, their total ticket count and their five most
// recent tickets.
func (uc *GetCustomerUseCase) Execute(ctx context.Context, orgID, customerID uuid.UUID) (*dto.CustomerDetail, error) {
	customer, err := uc.customerRepo.GetByID(ctx, orgID, customerID)
	if err != nil {
		return nil, err
	}

	count, err := uc.ticketRepo.CountByCustomer(ctx, orgID, customerID)
	if err != nil {
		return nil, err
	}

	id := customerID
	tickets, _, err := uc.ticketRepo.ListByOrg(ctx, orgID, repository.TicketFilter{
		CustomerID: &id,
		Sort:       repository.SortCreatedAtDesc,
	}, 0, recentTicketLimit)
	if err != nil {
		return nil, err
	}

	recent, err := uc.assembler.toListItems(ctx, orgID, tickets)
	if err != nil {
		return nil, err
	}

	return &dto.CustomerDetail{
		CustomerInfo:  toCustomerInfo(customer),
		TicketCount:   count,
		RecentTickets: recent,
	}, nil
}
