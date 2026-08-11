package triage

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ticketColumns = `t.id, t.organization_id, t.customer_id, t.department_id, t.assignee_id,
	t.subject, t.status, t.priority, t.priority_overridden, t.department_overridden,
	t.created_at, t.updated_at, t.resolved_at`

// priorityRank orders the priority enum by urgency rather than alphabetically.
const priorityRank = `CASE t.priority
	WHEN 'low' THEN 1 WHEN 'normal' THEN 2 WHEN 'high' THEN 3 WHEN 'urgent' THEN 4 ELSE 0 END`

// latestNeedsReview reads the needs_human_review flag of a ticket's most recent
// analysis, defaulting to false while classification is still pending.
const latestNeedsReview = `COALESCE((
	SELECT a.needs_human_review FROM ai_analyses a
	WHERE a.ticket_id = t.id ORDER BY a.created_at DESC LIMIT 1
), FALSE)`

// TicketRepo implements repository.TicketRepository with PostgreSQL.
type TicketRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.TicketRepository = (*TicketRepo)(nil)

// NewTicketRepo creates a new TicketRepo.
func NewTicketRepo(db *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{db: db}
}

func (r *TicketRepo) Create(ctx context.Context, ticket *model.Ticket) error {
	if ticket.ID == uuid.Nil {
		ticket.ID = uuid.New()
	}
	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now

	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO tickets (id, organization_id, customer_id, department_id, assignee_id,
			subject, status, priority, priority_overridden, department_overridden,
			created_at, updated_at, resolved_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		ticket.ID, ticket.OrganizationID, ticket.CustomerID, ticket.DepartmentID, ticket.AssigneeID,
		ticket.Subject, ticket.Status, ticket.Priority, ticket.PriorityOverridden,
		ticket.DepartmentOverridden, ticket.CreatedAt, ticket.UpdatedAt, ticket.ResolvedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create ticket", err)
	}
	return nil
}

func (r *TicketRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Ticket, error) {
	t, err := scanTicket(r.db.QueryRow(ctx,
		`SELECT `+ticketColumns+` FROM tickets t WHERE t.organization_id = $1 AND t.id = $2`,
		orgID, id,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "ticket not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get ticket", err)
	}
	return t, nil
}

func (r *TicketRepo) Update(ctx context.Context, ticket *model.Ticket) error {
	ticket.UpdatedAt = time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		`UPDATE tickets SET customer_id=$1, department_id=$2, assignee_id=$3, subject=$4,
			status=$5, priority=$6, priority_overridden=$7, department_overridden=$8,
			updated_at=$9, resolved_at=$10
		 WHERE organization_id=$11 AND id=$12`,
		ticket.CustomerID, ticket.DepartmentID, ticket.AssigneeID, ticket.Subject,
		ticket.Status, ticket.Priority, ticket.PriorityOverridden, ticket.DepartmentOverridden,
		ticket.UpdatedAt, ticket.ResolvedAt, ticket.OrganizationID, ticket.ID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update ticket", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "ticket not found", nil)
	}
	return nil
}

func (r *TicketRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM tickets WHERE organization_id=$1 AND id=$2`, orgID, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete ticket", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "ticket not found", nil)
	}
	return nil
}

func (r *TicketRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, filter repository.TicketFilter, offset, limit int) ([]*model.Ticket, int, error) {
	where, args := buildTicketWhere(orgID, filter)

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets t WHERE `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count tickets", err)
	}

	limitPos := "$" + strconv.Itoa(len(args)+1)
	offsetPos := "$" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx,
		`SELECT `+ticketColumns+` FROM tickets t WHERE `+where+
			` ORDER BY `+ticketOrderBy(filter.Sort)+
			` LIMIT `+limitPos+` OFFSET `+offsetPos,
		args...,
	)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list tickets", err)
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to scan ticket", err)
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to iterate tickets", err)
	}
	return tickets, total, nil
}

func (r *TicketRepo) CountByDepartment(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT department_id, COUNT(*) FROM tickets WHERE organization_id = $1 GROUP BY department_id`,
		orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count tickets by department", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var deptID uuid.UUID
		var count int
		if err := rows.Scan(&deptID, &count); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan department count", err)
		}
		counts[deptID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate department counts", err)
	}
	return counts, nil
}

func (r *TicketRepo) CountByCustomer(ctx context.Context, orgID, customerID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets WHERE organization_id = $1 AND customer_id = $2`,
		orgID, customerID,
	).Scan(&count)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to count tickets by customer", err)
	}
	return count, nil
}

func (r *TicketRepo) ReassignDepartment(ctx context.Context, orgID, fromDeptID, toDeptID uuid.UUID) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE tickets SET department_id=$1, updated_at=$2
		 WHERE organization_id=$3 AND department_id=$4`,
		toDeptID, time.Now().UTC(), orgID, fromDeptID,
	)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to reassign tickets", err)
	}
	return tag.RowsAffected(), nil
}

// buildTicketWhere turns a filter into a parameterised WHERE clause. Every value
// travels as a bind parameter; nothing from the caller reaches the SQL text.
func buildTicketWhere(orgID uuid.UUID, f repository.TicketFilter) (string, []any) {
	args := []any{orgID}
	where := []string{"t.organization_id = $1"}

	bind := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if len(f.Statuses) > 0 {
		statuses := make([]string, 0, len(f.Statuses))
		for _, s := range f.Statuses {
			statuses = append(statuses, string(s))
		}
		where = append(where, "t.status = ANY("+bind(statuses)+")")
	}

	if len(f.Priorities) > 0 {
		priorities := make([]string, 0, len(f.Priorities))
		for _, p := range f.Priorities {
			priorities = append(priorities, string(p))
		}
		where = append(where, "t.priority = ANY("+bind(priorities)+")")
	}

	if f.DepartmentID != nil {
		where = append(where, "t.department_id = "+bind(*f.DepartmentID))
	}

	// Unassigned wins over AssigneeID: ?assignee_id=unassigned is its own literal.
	switch {
	case f.Unassigned:
		where = append(where, "t.assignee_id IS NULL")
	case f.AssigneeID != nil:
		where = append(where, "t.assignee_id = "+bind(*f.AssigneeID))
	}

	if f.CustomerID != nil {
		where = append(where, "t.customer_id = "+bind(*f.CustomerID))
	}

	if f.NeedsReview != nil {
		where = append(where, latestNeedsReview+" = "+bind(*f.NeedsReview))
	}

	if f.Overridden != nil {
		cond := "(t.priority_overridden OR t.department_overridden)"
		if !*f.Overridden {
			cond = "NOT " + cond
		}
		where = append(where, cond)
	}

	if f.Query != "" {
		where = append(where, "t.subject ILIKE "+bind("%"+f.Query+"%"))
	}

	return strings.Join(where, " AND "), args
}

// ticketOrderBy maps the sort key onto a fixed ORDER BY fragment. Unknown values
// fall back to newest-first, so the caller's string is never interpolated.
func ticketOrderBy(sort string) string {
	switch sort {
	case repository.SortCreatedAtAsc:
		return "t.created_at ASC"
	case repository.SortPriorityAsc:
		return priorityRank + " ASC, t.created_at DESC"
	case repository.SortPriorityDesc:
		return priorityRank + " DESC, t.created_at DESC"
	case repository.SortUpdatedAtAsc:
		return "t.updated_at ASC"
	case repository.SortUpdatedAtDesc:
		return "t.updated_at DESC"
	default: // repository.SortCreatedAtDesc
		return "t.created_at DESC"
	}
}

func scanTicket(row rowScanner) (*model.Ticket, error) {
	var t model.Ticket
	if err := row.Scan(
		&t.ID, &t.OrganizationID, &t.CustomerID, &t.DepartmentID, &t.AssigneeID,
		&t.Subject, &t.Status, &t.Priority, &t.PriorityOverridden, &t.DepartmentOverridden,
		&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}
