package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the subset of pgx used by the repositories. Both *pgxpool.Pool and
// pgx.Tx satisfy it, so a repository written against DBTX runs identically on a
// pooled connection or inside an open transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// txContextKey carries an open transaction through the context so a repository
// can join a transaction it never learned about directly. Using the context
// keeps the repository method signatures unchanged.
type txContextKey struct{}

func contextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// Querier returns the transaction bound to ctx if there is one, otherwise the
// supplied fallback. A repository calls this instead of touching its pool
// directly so its writes automatically enrol in a surrounding TxManager.WithinTx.
func Querier(ctx context.Context, fallback DBTX) DBTX {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

// TxManager runs a function inside a single database transaction.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager creates a TxManager over the given pool.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithinTx begins a transaction, stores it on the context, and runs fn. The
// transaction commits when fn returns nil and rolls back otherwise, so a
// partial write can never survive an error.
//
// Calls nest safely: if ctx already carries a transaction, fn joins it rather
// than opening a second one, and the outermost WithinTx owns the commit.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(contextWithTx(ctx, tx)); err != nil {
		// Rollback failure is intentionally ignored: the original error is what
		// the caller needs, and a doomed transaction is discarded when the
		// connection returns to the pool regardless.
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
