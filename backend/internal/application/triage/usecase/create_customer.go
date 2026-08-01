package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// CreateCustomerUseCase handles customer creation.
type CreateCustomerUseCase struct {
	customerRepo repository.CustomerRepository
}

// NewCreateCustomerUseCase creates a new CreateCustomerUseCase.
func NewCreateCustomerUseCase(customerRepo repository.CustomerRepository) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{customerRepo: customerRepo}
}

// Execute creates a customer. Email is unique per organization.
func (uc *CreateCustomerUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.CreateCustomerRequest) (*dto.CustomerInfo, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if existing, _ := uc.customerRepo.GetByEmail(ctx, orgID, email); existing != nil {
		return nil, domainErr.New(domainErr.ErrAlreadyExists, "customer with this email already exists in this organization", nil)
	}

	customer := &model.Customer{
		OrganizationID: orgID,
		Email:          email,
		FullName:       strings.TrimSpace(req.FullName),
		Company:        strings.TrimSpace(req.Company),
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}

	info := toCustomerInfo(customer)
	return &info, nil
}

func toCustomerInfo(c *model.Customer) dto.CustomerInfo {
	return dto.CustomerInfo{
		ID:        c.ID,
		Email:     c.Email,
		FullName:  c.FullName,
		Company:   c.Company,
		CreatedAt: c.CreatedAt,
	}
}
