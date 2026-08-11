package iam

import (
	"context"
	"errors"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepo implements repository.RoleRepository with PostgreSQL.
type RoleRepo struct {
	db *pgxpool.Pool
}

// NewRoleRepo creates a new RoleRepo.
func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(ctx context.Context, role *model.Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now().UTC()
	role.CreatedAt = now
	role.UpdatedAt = now

	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO roles (id, scope_type, scope_id, name, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		role.ID, role.ScopeType, role.ScopeID, role.Name, role.Description, role.CreatedAt, role.UpdatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create role", err)
	}
	return nil
}

func (r *RoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	var role model.Role
	err := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT id, scope_type, scope_id, name, description, created_at, updated_at
		 FROM roles WHERE id = $1`, id,
	).Scan(&role.ID, &role.ScopeType, &role.ScopeID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "role not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get role", err)
	}
	return &role, nil
}

func (r *RoleRepo) ListByScope(ctx context.Context, scopeType model.ScopeType, scopeID uuid.UUID) ([]*model.Role, error) {
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT id, scope_type, scope_id, name, description, created_at, updated_at
		 FROM roles WHERE scope_type=$1 AND scope_id=$2 ORDER BY name`, scopeType, scopeID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list roles", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.ScopeType, &role.ScopeID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan role", err)
		}
		roles = append(roles, &role)
	}
	return roles, nil
}

func (r *RoleRepo) Update(ctx context.Context, role *model.Role) error {
	role.UpdatedAt = time.Now().UTC()
	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`UPDATE roles SET name=$1, description=$2, updated_at=$3 WHERE id=$4`,
		role.Name, role.Description, role.UpdatedAt, role.ID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to update role", err)
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Querier(ctx, r.db).Exec(ctx, `DELETE FROM roles WHERE id=$1`, id)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to delete role", err)
	}
	return nil
}

func (r *RoleRepo) AddPermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roleID, permission,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to add permission", err)
	}
	return nil
}

func (r *RoleRepo) RemovePermission(ctx context.Context, roleID uuid.UUID, permission string) error {
	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id=$1 AND permission=$2`,
		roleID, permission,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to remove permission", err)
	}
	return nil
}

func (r *RoleRepo) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT permission FROM role_permissions WHERE role_id=$1`, roleID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get permissions", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan permission", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *RoleRepo) AssignRoleToUser(ctx context.Context, userRole *model.UserRole) error {
	if userRole.ID == uuid.Nil {
		userRole.ID = uuid.New()
	}
	userRole.CreatedAt = time.Now().UTC()

	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO user_roles (id, user_id, role_id, organization_id, app_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, role_id, organization_id, app_id) DO NOTHING`,
		userRole.ID, userRole.UserID, userRole.RoleID, userRole.OrganizationID, userRole.AppID, userRole.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to assign role", err)
	}
	return nil
}

func (r *RoleRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to remove role from user", err)
	}
	return nil
}

func (r *RoleRepo) GetUserRoles(ctx context.Context, userID, orgID uuid.UUID) ([]*model.UserRole, error) {
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT id, user_id, role_id, organization_id, app_id, created_at
		 FROM user_roles WHERE user_id=$1 AND organization_id=$2`, userID, orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get user roles", err)
	}
	defer rows.Close()

	var userRoles []*model.UserRole
	for rows.Next() {
		var ur model.UserRole
		if err := rows.Scan(&ur.ID, &ur.UserID, &ur.RoleID, &ur.OrganizationID, &ur.AppID, &ur.CreatedAt); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan user role", err)
		}
		userRoles = append(userRoles, &ur)
	}
	return userRoles, nil
}

func (r *RoleRepo) GetUserRoleNames(ctx context.Context, userID, orgID uuid.UUID) ([]string, error) {
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT DISTINCT ro.name
		 FROM user_roles ur
		 JOIN roles ro ON ro.id = ur.role_id
		 WHERE ur.user_id = $1 AND ur.organization_id = $2
		 ORDER BY ro.name`, userID, orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get user role names", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan role name", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to iterate role names", err)
	}
	return names, nil
}

func (r *RoleRepo) GetPrimaryOrganization(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	// TODO(multi-org): the MVP assumes a user belongs to exactly one
	// organization, so the oldest assignment wins. Once multi-org membership is
	// supported this should return every organization and let the caller pick
	// the active one (e.g. from an X-Organization-ID header).
	var orgID uuid.UUID
	err := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT organization_id FROM user_roles WHERE user_id = $1
		 ORDER BY created_at ASC LIMIT 1`, userID,
	).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not an error: the user simply has no organization yet.
			return uuid.Nil, nil
		}
		return uuid.Nil, domainErr.New(domainErr.ErrInternal, "failed to resolve user organization", err)
	}
	return orgID, nil
}

func (r *RoleRepo) GetUserPermissions(ctx context.Context, userID, orgID uuid.UUID) ([]string, error) {
	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT DISTINCT rp.permission
		 FROM user_roles ur
		 JOIN role_permissions rp ON ur.role_id = rp.role_id
		 WHERE ur.user_id = $1 AND ur.organization_id = $2`, userID, orgID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get user permissions", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan permission", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}
