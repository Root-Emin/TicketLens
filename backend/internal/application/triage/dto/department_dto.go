package dto

import (
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
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
//
// Category deliberately carries no `oneof` tag, unlike the create request.
// `omitempty` tests a pointer for nil, not the string it points at, so a
// non-nil pointer to "" reached `oneof` and failed it — which made the documented
// way to clear a category unreachable: every PATCH with `"category": ""` came
// back 400 before UpdateDepartmentUseCase, which handles that exact case, ever
// ran. The use case is the authority on what a valid category is (it calls
// model.ValidCategory and answers 422), so the tag was redundant as well as
// wrong.
type UpdateDepartmentRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
}

// DepartmentInfo is the public department representation.
type DepartmentInfo struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    *model.Category `json:"category"`
	IsDefault   bool            `json:"is_default"`
	// TicketCount is every ticket ever routed here, not just the open ones.
	TicketCount int `json:"ticket_count"`
	// StaffCount is how many people are assigned to this department
	// (staff_departments, migration 00021). Portal customers are excluded, so
	// this always matches the length of GET /staff?department_id=<this>.
	StaffCount int `json:"staff_count"`
}
