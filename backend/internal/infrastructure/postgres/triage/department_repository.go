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

const departmentColumns = `id, organization_id, name, description, category, is_default, created_at, updated_at`

// DepartmentRepo implements repository.DepartmentRepository with PostgreSQL.
type DepartmentRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.DepartmentRepository = (*DepartmentRepo)(nil)

// NewDepartmentRepo creates a new DepartmentRepo.
func NewDepartmentRepo(db *pgxpool.Pool) *DepartmentRepo {
	return &DepartmentRepo{db: db}
}

func (r *DepartmentRepo) Create(ctx context.Context, department *model.Department) error {
	if department.ID == uuid.Nil {
		department.ID = uuid.New()
	}
	now := time.Now().UTC()
	department.CreatedAt = now
	department.UpdatedAt = now

	_, err := r.db.Exec(ctx,
		`INSERT INTO departments (id, organization_id, name, description, category, is_default, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		department.ID, department.OrganizationID, department.Name, department.Description,
		department.Category, department.IsDefault, department.CreatedAt, department.UpdatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create department", err)
	}
	return nil
}

func (r *DepartmentRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Department, error) {
	return r.getOne(ctx,
		`SELECT `+departmentColumns+` FROM departments WHERE organization_id = $1 AND id = $2`,
		"failed to get department", orgID, id,
	)
}

func (r *DepartmentRepo) GetByName(ctx context.Context, orgID uuid.UUID, name string) (*model.Department, error) {
	return r.getOne(ctx,
		`SELECT `+departmentColumns+` FROM departments WHERE organization_id = $1 AND name = $2`,
		"failed to get department by name", orgID, name,
	)
}

func (r *DepartmentRepo) GetByCategory(ctx context.Context, orgID uuid.UUID, category model.Category) (*model.Department, error) {
	return r.getOne(ctx,
		`SELECT `+departmentColumns+` FROM departments WHERE organization_id = $1 AND category = $2`,
		"failed to get department by category", orgID, string(category),
	)
}

func (r *DepartmentRepo) GetDefault(ctx context.Context, orgID uuid.UUID) (*model.Department, error) {
	return r.getOne(ctx,
		`SELECT `+departmentColumns+` FROM departments WHERE organization_id = $1 AND is_default`,
		"failed to get default department", orgID,
	)
}

func (r *DepartmentRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*model.Department, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+departmentColumns+`
		 FROM departments WHERE organization_id = $1 ORDER BY is_default DESC, name ASC`, orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list departments", err)
	}
	defer rows.Close()

	var departments []*model.Department
	for rows.Next() {
		d, err := scanDepartment(rows)
		if err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan department", err)
		}
		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate departments", err)
	}
	return departments, nil
}

func (r *DepartmentRepo) Update(ctx context.Context, department *model.Department) error {
	department.UpdatedAt = time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		`UPDATE departments SET name=$1, description=$2, category=$3, updated_at=$4
		 WHERE organization_id=$5 AND id=$6`,
		department.Name, department.Description, department.Category, department.UpdatedAt,
		department.OrganizationID, department.ID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update department", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "department not found", nil)
	}
	return nil
}

func (r *DepartmentRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM departments WHERE organization_id=$1 AND id=$2`, orgID, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete department", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "department not found", nil)
	}
	return nil
}

func (r *DepartmentRepo) getOne(ctx context.Context, query, failMsg string, args ...any) (*model.Department, error) {
	d, err := scanDepartment(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "department not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, failMsg, err)
	}
	return d, nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanDepartment(row rowScanner) (*model.Department, error) {
	var d model.Department
	var category *string
	if err := row.Scan(
		&d.ID, &d.OrganizationID, &d.Name, &d.Description, &category, &d.IsDefault, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if category != nil {
		c := model.Category(*category)
		d.Category = &c
	}
	return &d, nil
}
