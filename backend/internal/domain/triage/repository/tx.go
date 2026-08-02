package repository

import "context"

// TxManager runs a unit of work inside a single database transaction. The
// application layer depends on this interface; the postgres implementation lives
// in internal/shared/database.
//
// Writes performed by repositories on the context passed to fn commit together
// or not at all.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
