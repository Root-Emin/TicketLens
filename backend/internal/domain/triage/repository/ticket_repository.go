package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
)

// Sort keys accepted by TicketFilter.Sort. Anything else falls back to
// SortCreatedAtDesc; the value is never interpolated into SQL directly.
const (
	SortCreatedAtAsc  = "created_at"
	SortCreatedAtDesc = "-created_at"
	SortPriorityAsc   = "priority"
	SortPriorityDesc  = "-priority"
)

// TicketFilter holds the optional, combinable filters of the ticket queue.
// A nil pointer or empty slice means "do not filter on this field".
type TicketFilter struct {
	Statuses   []model.TicketStatus
	Priorities []model.TicketPriority

	DepartmentID *uuid.UUID
	AssigneeID   *uuid.UUID
	// Unassigned selects tickets with no assignee. It takes precedence over
	// AssigneeID and corresponds to the ?assignee_id=unassigned literal.
	Unassigned bool
	// CustomerID restricts the result to one customer's tickets, which is how
	// the ticket:read_own permission is enforced.
	CustomerID *uuid.UUID

	// NeedsReview filters on the needs_human_review flag of the ticket's most
	// recent analysis.
	NeedsReview *bool
	// Overridden selects tickets where a human corrected either AI prediction.
	Overridden *bool

	// Query is a case-insensitive substring match on the subject.
	Query string
	Sort  string
}

// TicketRepository defines the interface for ticket persistence.
type TicketRepository interface {
	Create(ctx context.Context, ticket *model.Ticket) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Ticket, error)
	Update(ctx context.Context, ticket *model.Ticket) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error

	ListByOrg(ctx context.Context, orgID uuid.UUID, filter TicketFilter, offset, limit int) ([]*model.Ticket, int, error)

	// CountByDepartment returns ticket counts keyed by department ID, used by
	// the department list without an N+1.
	CountByDepartment(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error)
	// CountByCustomer returns the number of tickets raised by one customer.
	CountByCustomer(ctx context.Context, orgID, customerID uuid.UUID) (int, error)
	// ReassignDepartment moves every ticket of one department to another and
	// returns how many rows moved. Used before deleting a department.
	ReassignDepartment(ctx context.Context, orgID, fromDeptID, toDeptID uuid.UUID) (int64, error)
}
