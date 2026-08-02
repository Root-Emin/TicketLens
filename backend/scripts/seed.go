package main

// seed.go - Database seeding script
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("🌱 Seeding database...")

	// Seed roles
	if err := seedRoles(ctx, db); err != nil {
		log.Fatalf("Failed to seed roles: %v", err)
	}

	fmt.Println("✅ Database seeded successfully!")
}

func seedRoles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{
			name:        "admin",
			description: "Full system administrator",
			permissions: []string{"*"},
		},
		{
			name:        "org_admin",
			description: "Organization administrator",
			permissions: []string{"org:*", "app:*", "user:*"},
		},
		{
			name:        "app_admin",
			description: "Application administrator",
			permissions: []string{"app:*", "endpoint:*"},
		},
		{
			name:        "developer",
			description: "Developer with read/write access",
			permissions: []string{"endpoint:read", "endpoint:write"},
		},
		{
			name:        "viewer",
			description: "Read-only access",
			permissions: []string{"*:read"},
		},
	}

	for _, r := range roles {
		roleID := uuid.New()
		// scope_id is the zero UUID: these are template roles, not bound to a
		// specific organization. The unique constraint is on the
		// (scope_type, scope_id, name) triple, so ON CONFLICT must match it.
		_, err := db.Exec(ctx, `
			INSERT INTO roles (id, scope_type, scope_id, name, description, created_at, updated_at)
			VALUES ($1, 'organization', $2, $3, $4, NOW(), NOW())
			ON CONFLICT (scope_type, scope_id, name) DO NOTHING
		`, roleID, uuid.Nil, r.name, r.description)
		if err != nil {
			return fmt.Errorf("insert role %s: %w", r.name, err)
		}

		// Insert permissions
		for _, perm := range r.permissions {
			_, err := db.Exec(ctx, `
				INSERT INTO role_permissions (role_id, permission, created_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT (role_id, permission) DO NOTHING
			`, roleID, perm)
			if err != nil {
				return fmt.Errorf("insert permission %s for role %s: %w", perm, r.name, err)
			}
		}

		fmt.Printf("  ✓ Seeded role: %s\n", r.name)
	}

	return nil
}
