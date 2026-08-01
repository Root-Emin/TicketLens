package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
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

// StatsRepository computes dashboard aggregates in SQL.
type StatsRepository interface {
	// Overview aggregates tickets created within [from, to) for one organization.
	Overview(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*StatsOverview, error)
}
