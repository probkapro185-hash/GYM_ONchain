package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type dbtx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type txContextKey struct{}

func queryer(ctx context.Context, pool *pgxpool.Pool) dbtx {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

type transactionManager struct{ pool *pgxpool.Pool }

func NewTransactionManager(pool *pgxpool.Pool) repository.TransactionManager {
	return &transactionManager{pool: pool}
}

func (m *transactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, alreadyInTx := ctx.Value(txContextKey{}).(pgx.Tx); alreadyInTx {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return fmt.Errorf("%w (rollback: %v)", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: duplicate value", domain.ErrAlreadyExists)
		case "23502", "23503", "23514", "22001", "22003", "22P02":
			return fmt.Errorf("%w: database constraint violation", domain.ErrInvalidInput)
		}
	}
	return err
}

func requireAffected(tag pgconn.CommandTag) error {
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
