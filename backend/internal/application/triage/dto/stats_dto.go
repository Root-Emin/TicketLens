package dto

import "github.com/google/uuid"

// TicketStats is the ticket count breakdown by status.
type TicketStats struct {
	Total           int `json:"total"`
	Open            int `json:"open"`
	InProgress      int `json:"in_progress"`
	PendingCustomer int `json:"pending_customer"`
	Resolved        int `json:"resolved"`
	Closed          int `json:"closed"`
}

// PriorityStats is the ticket count breakdown by priority.
type PriorityStats struct {
	Low    int `json:"low"`
	Normal int `json:"normal"`
	High   int `json:"high"`
	Urgent int `json:"urgent"`
}

// DepartmentStat is one row of the by-department breakdown.
type DepartmentStat struct {
	DepartmentID uuid.UUID `json:"department_id"`
	Name         string    `json:"name"`
	Count        int       `json:"count"`
}

// AIStatsResponse is the classifier scorecard.
//
// PriorityAcceptRate is the headline number of the project: tickets that have
// an analysis and were not corrected by a human, over tickets that have an
// analysis. It is 0 when nothing has been analyzed yet.
// DepartmentAcceptRate is computed only over analyses that were actually
// routed. UnmappedCount reports the rest: tickets whose predicted category has
// no department in this organization. A high unmapped_count with a healthy
// accept rate means the model is fine and the org needs to define departments.
type AIStatsResponse struct {
	Analyzed              int     `json:"analyzed"`
	PriorityAcceptRate    float64 `json:"priority_accept_rate"`
	DepartmentAcceptRate  float64 `json:"department_accept_rate"`
	AvgPriorityConfidence float64 `json:"avg_priority_confidence"`
	NeedsReview           int     `json:"needs_review"`
	UnmappedCount         int     `json:"unmapped_count"`
}

// StatsOverviewResponse is GET /stats/overview.
//
// ByCategory is the comparable signal: departments are per-organization, so
// by_department cannot be compared across tenants or against the model's own
// behaviour. Categories come from the fixed taxonomy, so they can. Every
// taxonomy label is present, zero included, to keep the shape stable.
type StatsOverviewResponse struct {
	Tickets      TicketStats      `json:"tickets"`
	ByPriority   PriorityStats    `json:"by_priority"`
	ByCategory   map[string]int   `json:"by_category"`
	ByDepartment []DepartmentStat `json:"by_department"`
	AI           AIStatsResponse  `json:"ai"`
}
