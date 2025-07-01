package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/jmoiron/sqlx"
)

func WithTx(ctx context.Context, reason string, fn func(tx *sqlx.Tx) error) error {
	return withTx(ctx, nil, reason, fn)
}

func WithTxOpts(ctx context.Context, opts *sql.TxOptions, reason string, fn func(tx *sqlx.Tx) error) error {
	return withTx(ctx, opts, reason, fn)
}

func withTx(ctx context.Context, opts *sql.TxOptions, reason string, fn func(tx *sqlx.Tx) error) error {
	log.Printf("starting transaction: (%s)", reason)

	tx, err := Conn.BeginTxx(ctx, opts)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	var committed bool

	// Ensure that rollback is attempted in case of failure
	defer func() {
		panicErr := recover()
		if panicErr != nil {
			log.Printf("panic in WithTx (%s): %v\n%s", reason, panicErr, debug.Stack())
			log.Printf("stack trace (panic - %s):\n%s", reason, debug.Stack())
		}

		if committed {
			return
		}

		if rbErr := tx.Rollback(); rbErr != nil {
			if rbErr == sql.ErrTxDone {
				log.Printf("attempted to roll back transaction, but it was already committed: (%s)", reason)
			} else {
				log.Printf("transaction rollback error: (%s) %v\n", reason, rbErr)
			}
		} else {
			log.Printf("transaction rolled back: (%s)", reason)
		}
	}()

	err = fn(tx)

	if err != nil {
		log.Printf("error in WithTx (%s): %v", reason, err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("error committing transaction: (%s) %v", reason, err)
		return fmt.Errorf("error committing transaction: %v", err)
	}

	committed = true

	log.Printf("committed transaction: (%s)", reason)

	return nil
}
