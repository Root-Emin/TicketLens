package triage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

const aiAnalysisColumns = `id, organization_id, ticket_id, predicted_priority, priority_confidence,
	predicted_category, predicted_department_id, department_confidence, needs_human_review,
	mapping_fallback, model_name, model_version, raw_response, created_at`

// AIAnalysisRepo implements repository.AIAnalysisRepository with PostgreSQL.
// The table is append-only, so there is no Update.
type AIAnalysisRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.AIAnalysisRepository = (*AIAnalysisRepo)(nil)

// NewAIAnalysisRepo creates a new AIAnalysisRepo.
func NewAIAnalysisRepo(db *pgxpool.Pool) *AIAnalysisRepo {
	return &AIAnalysisRepo{db: db}
}

func (r *AIAnalysisRepo) Create(ctx context.Context, analysis *model.AIAnalysis) error {
	if analysis.ID == uuid.Nil {
		analysis.ID = uuid.New()
	}
	analysis.CreatedAt = time.Now().UTC()

	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_analyses (id, organization_id, ticket_id, predicted_priority,
			priority_confidence, predicted_category, predicted_department_id, department_confidence,
			needs_human_review, mapping_fallback, model_name, model_version, raw_response, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		analysis.ID, analysis.OrganizationID, analysis.TicketID, analysis.PredictedPriority,
		analysis.PriorityConfidence, analysis.PredictedCategory, analysis.PredictedDepartmentID,
		analysis.DepartmentConfidence, analysis.NeedsHumanReview, analysis.MappingFallback,
		analysis.ModelName, analysis.ModelVersion, analysis.RawResponse, analysis.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create ai analysis", err)
	}
	return nil
}

func (r *AIAnalysisRepo) ListByTicket(ctx context.Context, orgID, ticketID uuid.UUID) ([]*model.AIAnalysis, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+aiAnalysisColumns+`
		 FROM ai_analyses WHERE organization_id = $1 AND ticket_id = $2
		 ORDER BY created_at DESC`,
		orgID, ticketID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list ai analyses", err)
	}
	defer rows.Close()

	var analyses []*model.AIAnalysis
	for rows.Next() {
		a, err := scanAIAnalysis(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan ai analysis", err)
		}
		analyses = append(analyses, a)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate ai analyses", err)
	}
	return analyses, nil
}

func (r *AIAnalysisRepo) GetLatestByTicket(ctx context.Context, orgID, ticketID uuid.UUID) (*model.AIAnalysis, error) {
	a, err := scanAIAnalysis(r.db.QueryRow(ctx,
		`SELECT `+aiAnalysisColumns+`
		 FROM ai_analyses WHERE organization_id = $1 AND ticket_id = $2
		 ORDER BY created_at DESC LIMIT 1`,
		orgID, ticketID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "ai analysis not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get latest ai analysis", err)
	}
	return a, nil
}

func (r *AIAnalysisRepo) LatestForTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]*model.AIAnalysis, error) {
	latest := make(map[uuid.UUID]*model.AIAnalysis)
	if len(ticketIDs) == 0 {
		return latest, nil
	}

	// DISTINCT ON keeps the newest row per ticket in a single pass.
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (ticket_id) `+aiAnalysisColumns+`
		 FROM ai_analyses WHERE organization_id = $1 AND ticket_id = ANY($2)
		 ORDER BY ticket_id, created_at DESC`,
		orgID, ticketIDs,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list latest ai analyses", err)
	}
	defer rows.Close()

	for rows.Next() {
		a, err := scanAIAnalysis(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan ai analysis", err)
		}
		latest[a.TicketID] = a
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate latest ai analyses", err)
	}
	return latest, nil
}

func scanAIAnalysis(row rowScanner) (*model.AIAnalysis, error) {
	var a model.AIAnalysis
	var category *string
	if err := row.Scan(
		&a.ID, &a.OrganizationID, &a.TicketID, &a.PredictedPriority, &a.PriorityConfidence,
		&category, &a.PredictedDepartmentID, &a.DepartmentConfidence, &a.NeedsHumanReview,
		&a.MappingFallback, &a.ModelName, &a.ModelVersion, &a.RawResponse, &a.CreatedAt,
	); err != nil {
		return nil, err
	}
	if category != nil {
		c := model.Category(*category)
		a.PredictedCategory = &c
	}
	return &a, nil
}
