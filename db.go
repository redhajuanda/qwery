package qwery

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type DB struct {
	*sqlx.DB
}

// getTx returns the transaction from the context if it exists.
// It returns an error if the transaction is not found in the context.
func (m *DB) getTx(ctx context.Context) (*sqlx.Tx, error) {

	if tx, ok := ctx.Value(contextKeyTx).(*sqlx.Tx); ok {
		return tx, nil
	}
	return nil, errors.New("failed to get transaction from context")

}

// Begin starts a new transaction in the PostgreSQL database.
// It takes a context.Context as input and returns a new context.Context and an error.
// The returned context.Context contains the transaction information that can be used in subsequent database operations.
// If an error occurs while starting the transaction, it returns nil and the error.
func (m *DB) Begin(ctx context.Context) (context.Context, error) {

	tx, err := m.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	// create and return a new context with the transaction information
	ctx = context.WithValue(ctx, contextKeyTx, tx)
	return ctx, nil

}

// Commit commits the current transaction.
// It retrieves the transaction from the context and calls the Commit method on it.
// If the transaction is not found in the context, it returns an error.
func (m *DB) Commit(ctx context.Context) error {

	tx, ok := ctx.Value(contextKeyTx).(*sqlx.Tx)
	if !ok {
		return errors.New("failed to commit, transaction not found in context")
	}

	err := tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil

}

// Rollback rolls back the transaction associated with the given context.
// It returns an error if the transaction is not found in the context.
// The rollback operation is performed using the pgx.Tx.Rollback method.
func (m *DB) Rollback(ctx context.Context) error {

	tx, ok := ctx.Value(contextKeyTx).(*sqlx.Tx)
	if !ok {
		return errors.New("failed to rollback, transaction not found in context")
	}

	err := tx.Rollback()
	if err != nil {
		return errors.Wrap(err, "failed to rollback transaction")
	}

	return nil

}
