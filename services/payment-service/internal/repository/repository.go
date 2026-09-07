package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
)

type PaymentRepository interface {
	GetBalance(userId string) (balance float64, err error)
	CreatePayment(userId string, isDebit bool, value float64) error
}

type PaymentRepositoryImpl struct {
	logger *slog.Logger
	db     *sql.DB
}

func NewPaymentRepository(db *sql.DB, logger *slog.Logger) PaymentRepository {
	return &PaymentRepositoryImpl{db: db, logger: logger}
}

func (p *PaymentRepositoryImpl) GetBalance(userId string) (balance float64, err error) {
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

func (p *PaymentRepositoryImpl) CreatePayment(userId string, isDebit bool, value float64) error {
	const query = `
		INSERT INTO balance_ledger (user_id, is_debit, amount)
		VALUES ($1, $2, $3)
	`

	_, err := p.db.Exec(query, userId, isDebit, value)
	if err != nil {
		p.logger.Error(err.Error())
		return fmt.Errorf("create payment for user %s: %w", userId, err)
	}

	return nil
}
