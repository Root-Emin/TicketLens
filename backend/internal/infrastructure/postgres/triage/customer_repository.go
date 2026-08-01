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

const customerColumns = `id, organization_id, email, full_name, company, created_at`

// CustomerRepo implements repository.CustomerRepository with PostgreSQL.
type CustomerRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.CustomerRepository = (*CustomerRepo)(nil)

// NewCustomerRepo creates a new CustomerRepo.
func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) Create(ctx context.Context, customer *model.Customer) error {
	if customer.ID == uuid.Nil {
		customer.ID = uuid.New()
	}
	customer.CreatedAt = time.Now().UTC()

	_, err := r.db.Exec(ctx,
		`INSERT INTO customers (id, organization_id, email, full_name, company, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		customer.ID, customer.OrganizationID, customer.Email, customer.FullName,
		customer.Company, customer.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create customer", err)
	}
	return nil
}

func (r *CustomerRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Customer, error) {
	return r.getOne(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE organization_id = $1 AND id = $2`,
		"failed to get customer", orgID, id,
	)
}

func (r *CustomerRepo) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Customer, error) {
	return r.getOne(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE organization_id = $1 AND email = $2`,
		"failed to get customer by email", orgID, email,
	)
}

func (r *CustomerRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, query string, offset, limit int) ([]*model.Customer, int, error) {
	// $2 is NULL when no search term was given, which disables both ILIKE terms.
	var search any
	if query != "" {
		search = "%" + query + "%"
	}

	const filter = `organization_id = $1
		 AND ($2::text IS NULL OR email ILIKE $2 OR full_name ILIKE $2)`

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM customers WHERE `+filter, orgID, search,
	).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count customers", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+customerColumns+` FROM customers WHERE `+filter+`
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		orgID, search, limit, offset,
	)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list customers", err)
	}
	defer rows.Close()

	var customers []*model.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to scan customer", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to iterate customers", err)
	}
	return customers, total, nil
}

func (r *CustomerRepo) ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*model.Customer, error) {
	byID := make(map[uuid.UUID]*model.Customer)
	if len(ids) == 0 {
		return byID, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+customerColumns+` FROM customers
		 WHERE organization_id = $1 AND id = ANY($2)`,
		orgID, ids,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list customers by id", err)
	}
	defer rows.Close()

	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan customer", err)
		}
		byID[c.ID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate customers", err)
	}
	return byID, nil
}

func (r *CustomerRepo) Update(ctx context.Context, customer *model.Customer) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE customers SET email=$1, full_name=$2, company=$3
		 WHERE organization_id=$4 AND id=$5`,
		customer.Email, customer.FullName, customer.Company,
		customer.OrganizationID, customer.ID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update customer", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "customer not found", nil)
	}
	return nil
}

func (r *CustomerRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM customers WHERE organization_id=$1 AND id=$2`, orgID, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete customer", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "customer not found", nil)
	}
	return nil
}

func (r *CustomerRepo) getOne(ctx context.Context, query, failMsg string, args ...any) (*model.Customer, error) {
	c, err := scanCustomer(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "customer not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, failMsg, err)
	}
	return c, nil
}

func scanCustomer(row rowScanner) (*model.Customer, error) {
	var c model.Customer
	if err := row.Scan(
		&c.ID, &c.OrganizationID, &c.Email, &c.FullName, &c.Company, &c.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}
