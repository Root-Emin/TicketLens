package triage

import (
	"context"
	"errors"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	_, err := database.Querier(ctx, r.db).Exec(ctx,
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

// PreviewByTickets returns the opening message of each ticket, truncated.
//
// DISTINCT ON is what keeps this one query: Postgres picks the first row per
// ticket_id under the given ORDER BY, so the whole page's previews come back in
// a single index scan rather than a query per ticket.
//
// Truncation happens in SQL so a 4000-character description is never carried
// over the wire just to be cut in Go. The ellipsis is added by the caller's
// display layer if it wants one; the value here is plain text.
func (r *TicketMessageRepo) PreviewByTickets(
	ctx context.Context,
	orgID uuid.UUID,
	ticketIDs []uuid.UUID,
	maxRunes int,
) (map[uuid.UUID]string, error) {
	previews := make(map[uuid.UUID]string)
	if len(ticketIDs) == 0 || maxRunes <= 0 {
		return previews, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (ticket_id) ticket_id, LEFT(body, $3)
		 FROM ticket_messages
		 WHERE organization_id = $1 AND ticket_id = ANY($2) AND is_internal = FALSE
		 ORDER BY ticket_id, created_at ASC`,
		orgID, ticketIDs, maxRunes,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to load ticket previews", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ticketID uuid.UUID
		var body string
		if err := rows.Scan(&ticketID, &body); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan ticket preview", err)
		}
		previews[ticketID] = body
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate ticket previews", err)
	}
	return previews, nil
}

// FirstResponseByTickets returns when support first answered each ticket.
//
// "Support" excludes the customer's own follow-ups and internal notes: a note
// the requester cannot see is not a response to them, and counting one would
// make the response-time metric flatter than the experience it describes.
func (r *TicketMessageRepo) FirstResponseByTickets(
	ctx context.Context,
	orgID uuid.UUID,
	ticketIDs []uuid.UUID,
) (map[uuid.UUID]time.Time, error) {
	responses := make(map[uuid.UUID]time.Time)
	if len(ticketIDs) == 0 {
		return responses, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT ticket_id, MIN(created_at)
		 FROM ticket_messages
		 WHERE organization_id = $1 AND ticket_id = ANY($2)
		   AND author_type <> 'customer' AND is_internal = FALSE
		 GROUP BY ticket_id`,
		orgID, ticketIDs,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to load first responses", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ticketID uuid.UUID
		var at time.Time
		if err := rows.Scan(&ticketID, &at); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan first response", err)
		}
		responses[ticketID] = at
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate first responses", err)
	}
	return responses, nil
}
