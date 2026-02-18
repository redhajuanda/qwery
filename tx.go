package qwery

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/redhajuanda/komon/logger"
)

// TxFunc represents the function signature for transaction callback.
type TxFunc func(tx *Tx) (out any, err error)

// Tx is a struct that used to run a transaction
type Tx struct {
	client *Client
	log    logger.Logger
	tx     *sqlx.Tx
}

// newTx returns a new transaction
func newTx(dclient *Client, log logger.Logger) *Tx {

	return &Tx{
		client: dclient,
		log:    log,
	}

}

// Run is a function to run query within the transaction
func (t *Tx) Run(runnerCode string) Runnerer {

	return newRunner(runnerParams{
		queryType:     queryTypeRunner,
		runnerCode:    runnerCode,
		client:        t.client,
		log:           t.log,
		inTransaction: true,
		tx:            t,
	})

}

// RunRaw is a function to run raw query within the transaction
func (t *Tx) RunRaw(query string) Runnerer {

	return newRunner(runnerParams{
		queryType:     queryTypeRaw,
		queryRaw:      query,
		client:        t.client,
		log:           t.log,
		inTransaction: true,
		tx:            t,
	})

}

// Begin starts a new transaction
func (t *Tx) Begin(ctx context.Context) error {
	tx, err := t.client.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	t.tx = tx
	return nil
}

// Commit commits the transaction
func (t *Tx) Commit() error {
	if t.tx == nil {
		return errors.New("no active transaction to commit")
	}
	err := t.tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	t.tx = nil
	return nil
}

// Rollback rolls back the transaction
func (t *Tx) Rollback() error {
	if t.tx == nil {
		return errors.New("no active transaction to rollback")
	}
	err := t.tx.Rollback()
	if err != nil {
		return errors.Wrap(err, "failed to rollback transaction")
	}
	t.tx = nil
	return nil
}

// GetTx returns the underlying sqlx.Tx for internal use
func (t *Tx) GetTx() *sqlx.Tx {
	return t.tx
}
