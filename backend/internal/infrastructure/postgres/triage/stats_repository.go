package triage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// StatsRepo implements repository.StatsRepository with PostgreSQL.
// Every aggregate is computed in SQL; nothing is counted in Go.
type StatsRepo struct {
	db *pgxpool.Pool
}

// NewStatsRepo creates a new StatsRepo.
func NewStatsRepo(db *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{db: db}
}

// Verify interface compliance at compile time.
var _ repository.StatsRepository = (*StatsRepo)(nil)

// window is the shared organization + created_at predicate. Tickets are counted
// by creation time, so [from, to) selects the reporting period.
const window = `organization_id = $1 AND created_at >= $2 AND created_at < $3`

func (r *StatsRepo) Overview(ctx context.Context, orgID uuid.UUID, from, to time.Time) (*repository.StatsOverview, error) {
	overview := &repository.StatsOverview{
		ByStatus:   make(map[model.TicketStatus]int),
		ByPriority: make(map[model.TicketPriority]int),
		ByCategory: make(map[model.Category]int),
	}

	// 1. Ticket counts per status (total is their sum).
	rows, err := r.db.Query(ctx,
		`SELECT status, COUNT(*) FROM tickets WHERE `+window+` GROUP BY status`,
		orgID, from, to,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count tickets by status", err)
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan status count", err)
		}
		overview.ByStatus[model.TicketStatus(status)] = count
		overview.Total += count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate status counts", err)
	}

	// 2. Ticket counts per priority.
	rows, err = r.db.Query(ctx,
		`SELECT priority, COUNT(*) FROM tickets WHERE `+window+` GROUP BY priority`,
		orgID, from, to,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count tickets by priority", err)
	}
	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err != nil {
			rows.Close()
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan priority count", err)
		}
		overview.ByPriority[model.TicketPriority(priority)] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate priority counts", err)
	}

	// 3. Ticket counts per department, including departments with no tickets.
	rows, err = r.db.Query(ctx,
		`SELECT d.id, d.name, COUNT(t.id)
		 FROM departments d
		 LEFT JOIN tickets t
		   ON t.department_id = d.id AND t.created_at >= $2 AND t.created_at < $3
		 WHERE d.organization_id = $1
		 GROUP BY d.id, d.name
		 ORDER BY COUNT(t.id) DESC, d.name ASC`,
		orgID, from, to,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count tickets by department", err)
	}
	for rows.Next() {
		var row repository.DepartmentTicketCount
		if err := rows.Scan(&row.DepartmentID, &row.Name, &row.Count); err != nil {
			rows.Close()
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan department count", err)
		}
		overview.ByDepartment = append(overview.ByDepartment, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate department counts", err)
	}

	// 4. Ticket counts per predicted category, one row per ticket via its most
	// recent analysis — a re-run must not make a ticket count twice.
	rows, err = r.db.Query(ctx,
		`WITH latest AS (
			SELECT DISTINCT ON (ticket_id) ticket_id, predicted_category
			FROM ai_analyses
			WHERE organization_id = $1
			ORDER BY ticket_id, created_at DESC
		)
		SELECT l.predicted_category, COUNT(*)
		FROM latest l
		JOIN tickets t ON t.id = l.ticket_id
		WHERE l.predicted_category IS NOT NULL AND t.`+window+`
		GROUP BY l.predicted_category`,
		orgID, from, to,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count tickets by category", err)
	}
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			rows.Close()
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan category count", err)
		}
		overview.ByCategory[model.Category(category)] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate category counts", err)
	}

	// 5. Classifier accuracy over the tickets that actually have an analysis.
	// DISTINCT ON picks each ticket's most recent analysis, then the ticket's
	// *_overridden flags say whether a human disagreed with it.
	//
	// The department columns filter on NOT mapping_fallback in BOTH the
	// numerator and the denominator: an analysis the application could not route
	// says nothing about whether the model was accepted.
	err = r.db.QueryRow(ctx,
		`WITH latest AS (
			SELECT DISTINCT ON (ticket_id) ticket_id, priority_confidence,
			       needs_human_review, mapping_fallback
			FROM ai_analyses
			WHERE organization_id = $1
			ORDER BY ticket_id, created_at DESC
		)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE NOT t.priority_overridden),
			COUNT(*) FILTER (WHERE NOT l.mapping_fallback),
			COUNT(*) FILTER (WHERE NOT l.mapping_fallback AND NOT t.department_overridden),
			COALESCE(AVG(l.priority_confidence), 0),
			COUNT(*) FILTER (WHERE l.needs_human_review),
			COUNT(*) FILTER (WHERE l.mapping_fallback)
		FROM latest l
		JOIN tickets t ON t.id = l.ticket_id
		WHERE t.`+window,
		orgID, from, to,
	).Scan(
		&overview.AI.Analyzed,
		&overview.AI.PriorityAccepted,
		&overview.AI.DepartmentAnalyzed,
		&overview.AI.DepartmentAccepted,
		&overview.AI.AvgPriorityConfidence,
		&overview.AI.NeedsReview,
		&overview.AI.Unmapped,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to aggregate ai stats", err)
	}

	return overview, nil
}
