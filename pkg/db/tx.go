package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ahwlsqja/StableCoin-B2B-Commerce-Settlement-Engine/internal/repository/db"
)

// TxRunner manages database transactions with sqlc Queries.
// Service layer uses this to maintain transaction boundaries while
// passing tx-bound Queries to repository operations.
type TxRunner struct {
	database *sql.DB
}

// NewTxRunner creates a new TxRunner instance.
func NewTxRunner(database *sql.DB) *TxRunner {
	return &TxRunner{database: database}
}

// WithTx executes the given function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
//
// Usage example:
//
//	err := txRunner.WithTx(ctx, func(q *db.Queries) error {
//	    // 1. Lock user row
//	    user, err := q.GetUserForUpdate(ctx, userID)
//	    if err != nil {
//	        return err
//	    }
//	    // 2. Clear existing primary wallet
//	    if err := q.ClearPrimaryWallet(ctx, userID); err != nil {
//	        return err
//	    }
//	    // 3. Set new primary wallet
//	    if _, err := q.SetWalletPrimary(ctx, params); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func (r *TxRunner) WithTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Create tx-bound Queries
	q := db.New(tx)

	// Execute the function
	if err := fn(q); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit on success
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// WithTxResult executes the given function within a database transaction
// and returns a result value. Useful when the transaction needs to return data.
//
// Usage example:
//
//	wallet, err := WithTxResult(ctx, txRunner, func(q *db.Queries) (*db.Wallet, error) {
//	    // Create wallet
//	    result, err := q.CreateWallet(ctx, params)
//	    if err != nil {
//	        return nil, err
//	    }
//	    id, _ := result.LastInsertId()
//	    return q.GetWalletByID(ctx, uint64(id))
//	})
func WithTxResult[T any](ctx context.Context, r *TxRunner, fn func(q *db.Queries) (T, error)) (T, error) {
	var result T

	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin transaction: %w", err)
	}

	q := db.New(tx)

	result, err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return result, fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return result, err
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

// Queries returns a non-transactional Queries instance.
// Use this for read-only operations that don't require transactions.
func (r *TxRunner) Queries() *db.Queries {
	return db.New(r.database)
}

// DB returns the underlying database connection.
// Use this sparingly - prefer using Queries() or WithTx().
func (r *TxRunner) DB() *sql.DB {
	return r.database
}
