package repository

import (
	"context"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// DepartmentTicketCount is one row of the by-department breakdown.
type DepartmentTicketCount struct {
	DepartmentID uuid.UUID
	Name         string
	Count        int
}

// AIStats aggregates how well the classifier is doing.
//
// Accepted counts follow the contract: a ticket counts as accepted when it has
// an analysis and the matching *_overridden flag is still false. Rates are left
// to the caller so the division stays in one place.
//
// The department figures have their own denominator. Analyses whose category
// could not be mapped to a department were never routed, so counting them as
// agreement would report a perfect score for an organization that defined no
// departments at all. They are excluded from both DepartmentAnalyzed and
// DepartmentAccepted, and surface separately as Unmapped.
type AIStats struct {
	Analyzed              int
	PriorityAccepted      int
	DepartmentAnalyzed    int
	DepartmentAccepted    int
	AvgPriorityConfidence float64
	NeedsReview           int
	Unmapped              int
}

// StatsOverview is the aggregated dashboard payload for one organization.
//
// ByCategory counts each ticket once, under the category of its most recent
// analysis. Tickets with no analysis are absent.
type StatsOverview struct {
	Total        int
	ByStatus     map[model.TicketStatus]int
	ByPriority   map[model.TicketPriority]int
	ByCategory   map[model.Category]int
	ByDepartment []DepartmentTicketCount
	AI           AIStats
}

// TicketSummary is the small aggregate the customer portal's dashboard needs.
//
// Separate from StatsOverview on purpose. Overview is the triage dashboard: AI
// accept rates, per-department breakdowns, review backlogs — all of it gated on
// stats:read, which a customer never holds. Handing a customer a filtered
// Overview would still compute (and risk leaking) figures about a queue that is
// none of their business. This carries only what someone can be told about
// their own requests.
//
// The averages are minutes and are nil when nothing has been measured yet: a
// customer with no resolved ticket has no average, and reporting zero would
// read as "answered instantly".
type TicketSummary struct {
	Total            int
	ByStatus         map[model.TicketStatus]int
	AvgFirstResponse *float64
	AvgResolution    *float64
}

// Open counts the tickets still being worked on.
func (s *TicketSummary) Open() int {
	return s.ByStatus[model.TicketStatusOpen] + s.ByStatus[model.TicketStatusInProgress]
}

// WaitingCustomer counts the tickets the requester has to answer.
func (s *TicketSummary) WaitingCustomer() int {
	return s.ByStatus[model.TicketStatusPendingCustomer]
}

// Resolved counts the tickets that are done, closed included.
func (s *TicketSummary) Resolved() int {
	return s.ByStatus[model.TicketStatusResolved] + s.ByStatus[model.TicketStatusClosed]
}

// StatsRepository computes dashboard aggregates in SQL.
type StatsRepository interface {
	// Overview aggregates tickets created within [from, to) for one organization.
	Overview(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*StatsOverview, error)

	// Summary aggregates one organization's tickets, or one customer's when
	// customerID is non-nil. The pointer is the scope: passing nil is the staff
	// view and is never reachable from a customer-scoped request.
	Summary(ctx context.Context, orgID uuid.UUID, customerID *uuid.UUID) (*TicketSummary, error)
}
