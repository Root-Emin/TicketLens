package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
)

// ListCustomersUseCase lists an organization's customers.
type ListCustomersUseCase struct {
	customerRepo repository.CustomerRepository
}

// NewListCustomersUseCase creates a new ListCustomersUseCase.
func NewListCustomersUseCase(customerRepo repository.CustomerRepository) *ListCustomersUseCase {
	return &ListCustomersUseCase{customerRepo: customerRepo}
}

// Execute returns a page of customers; query matches email or full name.
func (uc *ListCustomersUseCase) Execute(ctx context.Context, orgID uuid.UUID, query string, page dto.PageParams) (dto.ListResponse[dto.CustomerInfo], error) {
	customers, total, err := uc.customerRepo.ListByOrg(ctx, orgID, strings.TrimSpace(query), page.Offset(), page.Limit())
	if err != nil {
		return dto.ListResponse[dto.CustomerInfo]{}, err
	}

	items := make([]dto.CustomerInfo, 0, len(customers))
	for _, c := range customers {
		items = append(items, toCustomerInfo(c))
	}

	return dto.NewListResponse(items, page, total), nil
}
