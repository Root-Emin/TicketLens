package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
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

	existing, err := uc.customerRepo.GetByEmail(ctx, orgID, email)
	if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
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
