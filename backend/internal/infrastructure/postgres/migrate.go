package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateUp applies every pending goose-formatted migration in lexical order.
//
// Applied versions are recorded in schema_migrations so re-runs are no-ops.
// Only the `-- +goose Up` section of each file is executed. This keeps the
// server self-sufficient without requiring the goose CLI at boot.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	if pool == nil {
		return nil
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	applied := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("list applied migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		applied[name] = true
	}
	rows.Close()

	// Databases that were migrated by the older dev.sh / goose path have the
	// schema but an empty schema_migrations table. Re-running CREATE INDEX
	// statements without IF NOT EXISTS would fail, so treat an already-present
	// baseline table as "everything current is applied".
	if len(applied) == 0 {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'organizations'
			)`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			for _, name := range files {
				if _, err := pool.Exec(ctx,
					`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, name); err != nil {
					return fmt.Errorf("bootstrap schema_migrations with %s: %w", name, err)
				}
				applied[name] = true
			}
			if log != nil {
				log.Info("bootstrapped schema_migrations from existing schema", "count", len(files))
			}
		}
	}

	for _, name := range files {
		if applied[name] {
			continue
		}
		raw, err := migrationsFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		upSQL := extractGooseUp(string(raw))
		if strings.TrimSpace(upSQL) == "" {
			if log != nil {
				log.Warn("migration has empty Up section, skipping", "file", name)
			}
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		if log != nil {
			log.Info("applied migration", "file", name)
		}
	}
	return nil
}

func extractGooseUp(src string) string {
	const upMark = "-- +goose Up"
	const downMark = "-- +goose Down"

	upIdx := strings.Index(src, upMark)
	if upIdx < 0 {
		return src
	}
	body := src[upIdx+len(upMark):]
	if downIdx := strings.Index(body, downMark); downIdx >= 0 {
		body = body[:downIdx]
	}
	return strings.TrimSpace(body)
}
