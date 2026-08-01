package dto

import (
	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
)

// CreateDepartmentRequest is the input for creating a department.
//
// Category binds the department to one classifier label so predictions can be
// routed to it. At most one department per organization may claim a category.
type CreateDepartmentRequest struct {
	Name        string  `json:"name" validate:"required,min=2"`
	Description string  `json:"description"`
	Category    *string `json:"category" validate:"omitempty,oneof=technical_issue integration payment_ops billing onboarding how_to account_access feature_request sales compliance"`
}

// UpdateDepartmentRequest is the input for PATCH /departments/{id}.
// Fields are pointers so an omitted field is distinguishable from an empty one.
// Sending category as "" clears it.
type UpdateDepartmentRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Category    *string `json:"category" validate:"omitempty,oneof=technical_issue integration payment_ops billing onboarding how_to account_access feature_request sales compliance"`
}

// DepartmentInfo is the public department representation.
type DepartmentInfo struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    *model.Category `json:"category"`
	IsDefault   bool            `json:"is_default"`
	TicketCount int             `json:"ticket_count"`
}
