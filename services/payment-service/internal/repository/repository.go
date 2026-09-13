package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

type PaymentRepository interface {
	GetBalance(ctx context.Context, userId string) (balance float64, err error)
	CreatePayment(ctx context.Context, userId string, isDebit bool, value float64) error
}

type PaymentRepositoryImpl struct {
	logger *slog.Logger
	db     *sql.DB
}

func NewPaymentRepository(db *sql.DB, logger *slog.Logger) PaymentRepository {
	return &PaymentRepositoryImpl{db: db, logger: logger}
}

func (p *PaymentRepositoryImpl) GetBalance(ctx context.Context, userId string) (balance float64, err error) {
	const query = `
		SELECT COALESCE(SUM(CASE WHEN is_debit THEN amount ELSE -amount END), 0)
		FROM balance_ledger
		WHERE user_id = $1
	`

	err = p.db.QueryRow(query, userId).Scan(&balance)
	if err != nil {
		p.logger.Error(err.Error())
		return 0, fmt.Errorf("get balance for user %s: %w", userId, err)
	}

	return balance, nil
}

// repo
func (p *PaymentRepositoryImpl) CreatePayment(ctx context.Context, userId string, isDebit bool, amount float64) error {
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if !isDebit {
		var balance float64
		err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(CASE WHEN is_debit THEN amount ELSE -amount END), 0)
			FROM balance_ledger
			WHERE user_id = $1
			FOR UPDATE
		`, userId).Scan(&balance)
		if err != nil {
			return err
		}

		if balance < amount {
			return errors.New("insufficient balance")
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO balance_ledger (user_id, is_debit, amount)
		VALUES ($1, $2, $3)
	`, userId, isDebit, amount)
	if err != nil {
		return err
	}

	return tx.Commit()
}
