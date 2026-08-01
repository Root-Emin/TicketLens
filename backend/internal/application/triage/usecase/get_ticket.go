package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
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

// Execute fetches a ticket. The repository is organization-scoped, so a ticket
// belonging to another tenant surfaces as not-found rather than forbidden —
// the API must not leak that the row exists.
func (uc *GetTicketUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID, includeInternal bool) (*dto.TicketDetail, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	return uc.assembler.toDetail(ctx, orgID, ticket, includeInternal)
}
