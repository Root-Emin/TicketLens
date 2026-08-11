// Command migrate applies the database migrations embedded in this build and
// exits.
//
// It exists so that schema changes are a deploy stage of their own rather than
// something the server does on the way up: the compose stack runs this to
// completion before any server container starts, and a failure here stops the
// rollout with the old version still serving. It reads exactly the same
// environment as cmd/server and shares its migration code, so the two can never
// disagree about what "applied" means.
//
//	migrate            apply pending migrations
//	migrate -status    list pending migrations and exit without applying
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Root-Emin/TicketLens/internal/infrastructure/postgres"
	"github.com/Root-Emin/TicketLens/internal/shared/config"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	"github.com/Root-Emin/TicketLens/internal/shared/logger"
	"github.com/Root-Emin/TicketLens/internal/shared/version"
)

// connectTimeout bounds reaching the database; migrationTimeout bounds the DDL
// itself, which can run long against a table that already holds data.
const (
	connectTimeout   = 30 * time.Second
	migrationTimeout = 10 * time.Minute
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	statusOnly := flag.Bool("status", false, "list pending migrations without applying them")
	flag.Parse()

	cfg := config.Load()
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	log.Info("ticketlens migrate",
		"version", version.Version,
		"commit", version.Commit,
		"env", cfg.Env,
		"db_host", cfg.Database.Host,
		"db_name", cfg.Database.DBName,
	)

	// Validate for the same reason the server does: a migration run carries the
	// database credentials, and a stack still holding development defaults is
	// one that should be fixed before it touches a production schema.
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), connectTimeout)
	defer cancelConnect()

	db, err := database.NewPostgresPool(connectCtx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	pending, err := postgres.PendingMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("check pending migrations: %w", err)
	}

	if len(pending) == 0 {
		log.Info("schema is up to date, nothing to apply")
		return nil
	}

	if *statusOnly {
		log.Info("pending migrations", "count", len(pending))
		for _, name := range pending {
			fmt.Println(name)
		}
		return nil
	}

	log.Info("applying migrations", "count", len(pending), "first", pending[0])
	if err := postgres.MigrateUp(ctx, db, log); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	log.Info("migrations applied", "count", len(pending))
	return nil
}
