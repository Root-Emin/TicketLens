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

const ticketMessageColumns = `id, organization_id, ticket_id, author_type, author_id,
	body, is_internal, created_at`

// TicketMessageRepo implements repository.TicketMessageRepository with PostgreSQL.
type TicketMessageRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.TicketMessageRepository = (*TicketMessageRepo)(nil)

// NewTicketMessageRepo creates a new TicketMessageRepo.
func NewTicketMessageRepo(db *pgxpool.Pool) *TicketMessageRepo {
	return &TicketMessageRepo{db: db}
}

func (r *TicketMessageRepo) Create(ctx context.Context, message *model.TicketMessage) error {
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}
	message.CreatedAt = time.Now().UTC()

	_, err := r.db.Exec(ctx,
		`INSERT INTO ticket_messages (id, organization_id, ticket_id, author_type, author_id,
			body, is_internal, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		message.ID, message.OrganizationID, message.TicketID, message.AuthorType,
		message.AuthorID, message.Body, message.IsInternal, message.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create ticket message", err)
	}
	return nil
}

func (r *TicketMessageRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.TicketMessage, error) {
	m, err := scanTicketMessage(r.db.QueryRow(ctx,
		`SELECT `+ticketMessageColumns+`
		 FROM ticket_messages WHERE organization_id = $1 AND id = $2`,
		orgID, id,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "ticket message not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get ticket message", err)
	}
	return m, nil
}

func (r *TicketMessageRepo) ListByTicket(ctx context.Context, orgID, ticketID uuid.UUID, includeInternal bool) ([]*model.TicketMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+ticketMessageColumns+`
		 FROM ticket_messages
		 WHERE organization_id = $1 AND ticket_id = $2 AND ($3 OR NOT is_internal)
		 ORDER BY created_at ASC`,
		orgID, ticketID, includeInternal,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list ticket messages", err)
	}
	defer rows.Close()

	var messages []*model.TicketMessage
	for rows.Next() {
		m, err := scanTicketMessage(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan ticket message", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate ticket messages", err)
	}
	return messages, nil
}

func (r *TicketMessageRepo) CountByTickets(ctx context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int)
	if len(ticketIDs) == 0 {
		return counts, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT ticket_id, COUNT(*) FROM ticket_messages
		 WHERE organization_id = $1 AND ticket_id = ANY($2)
		 GROUP BY ticket_id`,
		orgID, ticketIDs,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count ticket messages", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ticketID uuid.UUID
		var count int
		if err := rows.Scan(&ticketID, &count); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan message count", err)
		}
		counts[ticketID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate message counts", err)
	}
	return counts, nil
}

func scanTicketMessage(row rowScanner) (*model.TicketMessage, error) {
	var m model.TicketMessage
	if err := row.Scan(
		&m.ID, &m.OrganizationID, &m.TicketID, &m.AuthorType, &m.AuthorID,
		&m.Body, &m.IsInternal, &m.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &m, nil
}
