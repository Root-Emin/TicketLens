package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
)

// defaultStatsWindow is the reporting period when no from/to is supplied.
const defaultStatsWindow = 30 * 24 * time.Hour

// StatsOverviewUseCase builds the dashboard payload.
type StatsOverviewUseCase struct {
	statsRepo repository.StatsRepository
}

// NewStatsOverviewUseCase creates a new StatsOverviewUseCase.
func NewStatsOverviewUseCase(statsRepo repository.StatsRepository) *StatsOverviewUseCase {
	return &StatsOverviewUseCase{statsRepo: statsRepo}
}

// Execute aggregates the organization's tickets over [from, to). Zero values
// default to the last 30 days.
func (uc *StatsOverviewUseCase) Execute(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*dto.StatsOverviewResponse, error) {
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.Add(-defaultStatsWindow)
	}

	overview, err := uc.statsRepo.Overview(ctx, orgID, from, to)
	if err != nil {
		return nil, err
	}

	// Zero-fill the taxonomy so the payload shape never depends on which
	// categories happen to have tickets.
	categories := make(map[string]int, len(model.AllCategories))
	for _, category := range model.AllCategories {
		categories[string(category)] = overview.ByCategory[category]
	}

	departments := make([]dto.DepartmentStat, 0, len(overview.ByDepartment))
	for _, d := range overview.ByDepartment {
		departments = append(departments, dto.DepartmentStat{
			DepartmentID: d.DepartmentID,
			Name:         d.Name,
			Count:        d.Count,
		})
	}

	return &dto.StatsOverviewResponse{
		Tickets: dto.TicketStats{
			Total:           overview.Total,
			Open:            overview.ByStatus[model.TicketStatusOpen],
			InProgress:      overview.ByStatus[model.TicketStatusInProgress],
			PendingCustomer: overview.ByStatus[model.TicketStatusPendingCustomer],
			Resolved:        overview.ByStatus[model.TicketStatusResolved],
			Closed:          overview.ByStatus[model.TicketStatusClosed],
		},
		ByPriority: dto.PriorityStats{
			Low:    overview.ByPriority[model.TicketPriorityLow],
			Normal: overview.ByPriority[model.TicketPriorityNormal],
			High:   overview.ByPriority[model.TicketPriorityHigh],
			Urgent: overview.ByPriority[model.TicketPriorityUrgent],
		},
		ByCategory:   categories,
		ByDepartment: departments,
		AI: dto.AIStatsResponse{
			Analyzed:           overview.AI.Analyzed,
			PriorityAcceptRate: rate(overview.AI.PriorityAccepted, overview.AI.Analyzed),
			// Priority prediction is independent of routing, so it keeps the full
			// denominator; the department rate only counts analyses that were
			// actually mapped to a department.
			DepartmentAcceptRate:  rate(overview.AI.DepartmentAccepted, overview.AI.DepartmentAnalyzed),
			AvgPriorityConfidence: overview.AI.AvgPriorityConfidence,
			NeedsReview:           overview.AI.NeedsReview,
			UnmappedCount:         overview.AI.Unmapped,
		},
	}, nil
}

// rate divides accepted by analyzed, returning 0 when nothing was analyzed
// rather than NaN.
func rate(accepted, analyzed int) float64 {
	if analyzed == 0 {
		return 0
	}
	return float64(accepted) / float64(analyzed)
}
