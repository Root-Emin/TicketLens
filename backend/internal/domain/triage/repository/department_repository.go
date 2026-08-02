package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// DepartmentRepository defines the interface for department persistence.
//
// Every lookup takes an organization ID: a read for another tenant must come
// back as "not found" rather than leaking the row's existence.
type DepartmentRepository interface {
	Create(ctx context.Context, department *model.Department) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Department, error)
	GetByName(ctx context.Context, orgID uuid.UUID, name string) (*model.Department, error)
	// GetByCategory returns the department claiming a classifier category, or
	// ErrNotFound when the organization has none for it.
	GetByCategory(ctx context.Context, orgID uuid.UUID, category model.Category) (*model.Department, error)
	// GetDefault returns the organization's default department — the fallback
	// for predictions with no matching department.
	GetDefault(ctx context.Context, orgID uuid.UUID) (*model.Department, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*model.Department, error)
	Update(ctx context.Context, department *model.Department) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}
