package main

// seed.go - Template-role seeding script (roles only, not demo data).
// Prefer: go run ./cmd/seed  for a full demo tenant.
// Run with: go run scripts/seed.go

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Root-Emin/TicketLens/internal/shared/config"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("🌱 Seeding template roles...")

	if err := seedRoles(ctx, db); err != nil {
		log.Fatalf("Failed to seed roles: %v", err)
	}

	fmt.Println("✅ Template roles seeded. Re-sign in and wait out Redis permission cache (≤15m) if APIs still 403.")
}

// Permission matrix mirrors cmd/seed templateRoleDefs / frontend ROLE_PERMISSIONS.
func seedRoles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{"admin", "Full system administrator", []string{"*"}},
		{"org_admin", "Organization administrator", []string{
			"org:*", "app:*", "user:*",
			"ticket:create", "ticket:read", "ticket:update", "ticket:delete", "ticket:assign",
			"message:create", "department:manage", "customer:manage", "analysis:read", "stats:read",
		}},
		{"app_admin", "Application administrator", []string{"app:*", "endpoint:*"}},
		{"developer", "Developer with read/write access", []string{"endpoint:read", "endpoint:write"}},
		{"viewer", "Read-only access", []string{"*:read"}},
		{"agent", "Support agent", []string{
			"ticket:create", "ticket:read", "ticket:update", "ticket:assign",
			"message:create", "customer:manage", "analysis:read",
		}},
		{"customer", "Portal customer", []string{
			"ticket:create", "ticket:read_own", "ticket:reopen_own", "message:create",
		}},
	}

	for _, r := range roles {
		var roleID uuid.UUID
		err := db.QueryRow(ctx, `
			INSERT INTO roles (id, scope_type, scope_id, name, description, created_at, updated_at)
			VALUES ($1, 'organization', $2, $3, $4, NOW(), NOW())
			ON CONFLICT (scope_type, scope_id, name)
			DO UPDATE SET description = EXCLUDED.description, updated_at = NOW()
			RETURNING id
		`, uuid.New(), uuid.Nil, r.name, r.description).Scan(&roleID)
		if err != nil {
			return fmt.Errorf("upsert role %s: %w", r.name, err)
		}

		// Exact set: add missing, drop stale (including every org clone of this name).
		rows, err := db.Query(ctx, `SELECT id FROM roles WHERE name = $1`, r.name)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()

		for _, id := range ids {
			if err := replacePermissions(ctx, db, id, r.permissions); err != nil {
				return fmt.Errorf("permissions for %s (%s): %w", r.name, id, err)
			}
		}

		fmt.Printf("  ✓ Seeded role: %s (%d instances)\n", r.name, len(ids))
	}

	return nil
}

func replacePermissions(ctx context.Context, db *pgxpool.Pool, roleID uuid.UUID, desired []string) error {
	rows, err := db.Query(ctx, `SELECT permission FROM role_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}
	current := map[string]struct{}{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		current[p] = struct{}{}
	}
	rows.Close()

	want := map[string]struct{}{}
	for _, p := range desired {
		want[p] = struct{}{}
		if _, ok := current[p]; ok {
			continue
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (role_id, permission) DO NOTHING
		`, roleID, p); err != nil {
			return err
		}
	}
	for p := range current {
		if _, ok := want[p]; ok {
			continue
		}
		if _, err := db.Exec(ctx, `
			DELETE FROM role_permissions WHERE role_id = $1 AND permission = $2
		`, roleID, p); err != nil {
			return err
		}
	}
	return nil
}
