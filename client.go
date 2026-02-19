package qwery

import (
	"context"
	"database/sql"

	"github.com/redhajuanda/komon/cache"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/qwery/parser"

	"github.com/pkg/errors"
)

type Runable interface {
	Run(runner string) Runnerer
	RunRaw(query string) Runnerer
}

// Client is the main struct for the qwery client.
// It contains the database connection, runners, placeholder format, and logger.
// It provides methods to run queries and manage transactions.
type Client struct {
	db          *DB
	runners     map[string]string
	placeholder parser.Placeholder
	log         logger.Logger
	cache       cache.Cache
}

// Run initializes a new Runner with the given runner code.
func (c *Client) Run(runner string) Runnerer {

	return newRunner(runnerParams{
		queryType:     queryTypeRunner,
		runnerCode:    runner,
		client:        c,
		log:           c.log,
		inTransaction: false,
		tx:            nil,
	})

}

// RunRaw initializes a new Runner with the given raw query.
func (c *Client) RunRaw(query string) Runnerer {

	return newRunner(runnerParams{
		queryType:     queryTypeRaw,
		queryRaw:      query,
		client:        c,
		log:           c.log,
		inTransaction: false,
		tx:            nil,
	})
}

func (c *Client) DB() *sql.DB {
	return c.db.DB.DB
}

// WithTransaction initializes a new query with transaction.
// it takes a context and callback as input.
// callback is a function that will be executed in the transaction.
// callback takes tx as input.
// tx is a struct that contains the transaction configs.
func (c *Client) WithTransaction(ctx context.Context, callback TxFunc) (out any, err error) {

	// create new transaction object
	tx := newTx(c, c.log)

	// begin transaction
	c.log.WithContext(ctx).Debug("beginning transaction")
	err = tx.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	// defer rollback or commit transaction
	// if panic occurs, rollback transaction
	// if error occurs, rollback transaction
	// if no panic or error occurs, commit transaction
	defer c.handleTransactionWithTx(ctx, tx, &err)

	// execute callback
	c.log.WithContext(ctx).Debug("executing callback")
	out, err = callback(tx)

	return

}

func (c *Client) BeginTransaction(ctx context.Context) (*Tx, error) {

	tx := newTx(c, c.log)
	err := tx.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	return tx, nil

}

// CommitTransaction commits a transaction using the tx object
// Deprecated: Use tx.Commit() directly instead
func (c *Client) CommitTransaction(tx *Tx) error {
	return tx.Commit()
}

// RollbackTransaction rolls back a transaction using the tx object
// Deprecated: Use tx.Rollback() directly instead
func (c *Client) RollbackTransaction(tx *Tx) error {
	return tx.Rollback()
}

// handleTransactionWithTx handles the transaction logic using the tx object.
// It rolls back the transaction if a panic occurs or if an error is passed as input.
// If no panic or error occurs, it commits the transaction.
// It returns an error if there is a failure in rolling back or committing the transaction.
func (c *Client) handleTransactionWithTx(ctx context.Context, tx *Tx, errIn *error) (errOut error) {

	if p := recover(); p != nil {

		c.log.WithContext(ctx).Debug("panic occurred, rolling back transaction")

		err := tx.Rollback()
		if err != nil {
			errOut = errors.Wrap(err, "failed to rollback transaction")
		}
		panic(p) // re-throw panic after Rollback

	} else if *errIn != nil {

		c.log.WithContext(ctx).Debug("error occurred, rolling back transaction")

		err := tx.Rollback()
		if err != nil {
			errOut = errors.Wrap(err, "failed to rollback transaction")
		}

	} else {

		c.log.WithContext(ctx).Debug("committing transaction")

		err := tx.Commit()
		if err != nil {
			errOut = errors.Wrap(err, "failed to commit transaction")
		}

	}
	return

}
