package repository

import (
	"database/sql"
	"fmt"
)

type PaymentRepository interface {
	GetBalance(userId string) (balance float64, err error)
	CreatePayment(userId string, isDebit bool, value float64) error
}

type PaymentRepositoryImpl struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &PaymentRepositoryImpl{db: db}
}

func (r *PaymentRepositoryImpl) GetBalance(userId string) (balance float64, err error) {
	const query = `
		SELECT COALESCE(SUM(CASE WHEN is_debit THEN amount ELSE -amount END), 0)
		FROM balance_ledger
		WHERE user_id = $1
	`

	err = r.db.QueryRow(query, userId).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("get balance for user %s: %w", userId, err)
	}

	return balance, nil
}

func (p PaymentRepositoryImpl) CreatePayment(userId string, isDebit bool, value float64) error {
	const query = `
		INSERT INTO balance_ledger (user_id, is_debit, amount)
		VALUES ($1, $2, $3)
	`

	_, err := p.db.Exec(query, userId, isDebit, value)
	if err != nil {
		return fmt.Errorf("create payment for user %s: %w", userId, err)
	}

	return nil
}
