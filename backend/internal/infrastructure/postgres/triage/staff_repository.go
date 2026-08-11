package triage

import (
	"context"
	"errors"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
	The roster query, once.

	Four joins, and each earns its place:

	  user_roles         is where membership of an organization actually lives.
	                     organization_users looks like the right table and is
	                     never written (shared/middleware/org_scope.go says so),
	                     so joining it would return an empty roster.

	  customers          is the exclusion. A portal customer with a login holds a
	                     role in the organization exactly like a colleague does,
	                     so GET /users returns both and nothing on the response
	                     tells them apart. customers.user_id (migration 00020) is
	                     the discriminator, and `c.id IS NULL` is what keeps
	                     three shop owners out of the support headcount.

	  staff_departments  is the assignment. LEFT, because being on no team is a
	                     normal state and an inner join would hide every
	                     unassigned person — the exact people a manager opens
	                     this screen to find.

	  departments        carries the name, so the caller does not have to make a
	                     second round trip to render a roster.

	DISTINCT is required: a user holding two roles in one organization has two
	user_roles rows and would otherwise appear on the roster twice.
*/

const staffColumns = `DISTINCT u.id, u.email, u.first_name, u.last_name, u.status,
	u.created_at, sd.department_id, COALESCE(d.name, '')`

const staffFrom = `FROM users u
	JOIN user_roles ur ON ur.user_id = u.id AND ur.organization_id = $1
	LEFT JOIN customers c ON c.user_id = u.id AND c.organization_id = $1
	LEFT JOIN staff_departments sd ON sd.user_id = u.id AND sd.organization_id = $1
	LEFT JOIN departments d ON d.id = sd.department_id
	WHERE c.id IS NULL`

// StaffRepo implements repository.StaffRepository with PostgreSQL.
type StaffRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.StaffRepository = (*StaffRepo)(nil)

// NewStaffRepo creates a new StaffRepo.
func NewStaffRepo(db *pgxpool.Pool) *StaffRepo {
	return &StaffRepo{db: db}
}

func (r *StaffRepo) ListByOrg(
	ctx context.Context,
	orgID uuid.UUID,
	filter repository.StaffFilter,
	offset, limit int,
) ([]*model.StaffMember, int, error) {
	// $2 and $3 are NULL when unset, which disables their predicates. Written
	// this way rather than by concatenating clauses so the parameter positions
	// are fixed and no caller value is ever interpolated into SQL.
	var search any
	if filter.Query != "" {
		search = "%" + filter.Query + "%"
	}

	var departmentID any
	if filter.DepartmentID != nil && !filter.Unassigned {
		departmentID = *filter.DepartmentID
	}

	where := staffFrom + `
		AND ($2::text IS NULL
		     OR u.email ILIKE $2
		     OR COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') ILIKE $2)
		AND ($3::uuid IS NULL OR sd.department_id = $3)`

	// Unassigned is its own predicate rather than "department_id = NULL",
	// which no comparison can ever satisfy.
	if filter.Unassigned {
		where += ` AND sd.department_id IS NULL`
	}

	var total int
	if err := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT COUNT(*) FROM (SELECT DISTINCT u.id `+where+`) AS matched`,
		orgID, search, departmentID,
	).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count staff", err)
	}

	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT `+staffColumns+` `+where+`
		 ORDER BY u.created_at DESC LIMIT $4 OFFSET $5`,
		orgID, search, departmentID, limit, offset,
	)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list staff", err)
	}
	defer rows.Close()

	var staff []*model.StaffMember
	for rows.Next() {
		member, err := scanStaff(rows, orgID)
		if err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to scan staff member", err)
		}
		staff = append(staff, member)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to iterate staff", err)
	}
	return staff, total, nil
}

func (r *StaffRepo) GetByUser(ctx context.Context, orgID, userID uuid.UUID) (*model.StaffMember, error) {
	member, err := scanStaff(
		database.Querier(ctx, r.db).QueryRow(ctx, `SELECT `+staffColumns+` `+staffFrom+` AND u.id = $2`, orgID, userID),
		orgID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "staff member not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get staff member", err)
	}
	return member, nil
}

func (r *StaffRepo) SetDepartment(ctx context.Context, orgID, userID uuid.UUID, departmentID *uuid.UUID) error {
	if departmentID == nil {
		// Removing an assignment that was never there is not an error: the
		// caller asked for "on no team" and that is now true.
		if _, err := database.Querier(ctx, r.db).Exec(ctx,
			`DELETE FROM staff_departments WHERE organization_id=$1 AND user_id=$2`,
			orgID, userID,
		); err != nil {
			return domainErr.New(domainErr.ErrInternal, "failed to unassign staff member", err)
		}
		return nil
	}

	// The upsert is what makes reassignment idempotent — a manager moving
	// somebody who is already on the target team gets a no-op rather than a
	// duplicate key error.
	if _, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO staff_departments (organization_id, user_id, department_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (organization_id, user_id)
		 DO UPDATE SET department_id = EXCLUDED.department_id, updated_at = NOW()`,
		orgID, userID, *departmentID,
	); err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to assign staff member", err)
	}
	return nil
}

func (r *StaffRepo) CountByDepartment(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	// Counts the roster rather than the assignment table directly: a customer
	// could in principle hold a staff_departments row, and the headcount beside
	// a department has to mean the same thing as the list behind it.
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT sd.department_id, COUNT(DISTINCT u.id)
		 FROM staff_departments sd
		 JOIN users u ON u.id = sd.user_id
		 JOIN user_roles ur ON ur.user_id = u.id AND ur.organization_id = $1
		 LEFT JOIN customers c ON c.user_id = u.id AND c.organization_id = $1
		 WHERE sd.organization_id = $1 AND c.id IS NULL
		 GROUP BY sd.department_id`,
		orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to count staff by department", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var departmentID uuid.UUID
		var count int
		if err := rows.Scan(&departmentID, &count); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan staff count", err)
		}
		counts[departmentID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate staff counts", err)
	}
	return counts, nil
}

func scanStaff(row rowScanner, orgID uuid.UUID) (*model.StaffMember, error) {
	member := model.StaffMember{OrganizationID: orgID}

	// first_name and last_name are nullable (migration 00002), so they are
	// scanned through pointers rather than straight into the strings.
	var firstName, lastName *string

	if err := row.Scan(
		&member.UserID, &member.Email, &firstName, &lastName, &member.Status,
		&member.CreatedAt, &member.DepartmentID, &member.DepartmentName,
	); err != nil {
		return nil, err
	}

	if firstName != nil {
		member.FirstName = *firstName
	}
	if lastName != nil {
		member.LastName = *lastName
	}
	return &member, nil
}
