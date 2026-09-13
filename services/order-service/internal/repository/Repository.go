package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"order-service/internal/model"
	orderv1 "simple-marketplace-project/gen/order/v1"

	"github.com/google/uuid"
)

type OrderRepository interface {
	GetOrderProducts(orderId uuid.UUID) ([]model.OrderProduct, error)
	GetOrderState(orderId uuid.UUID) (string, error)
	CreateOrder(products []model.OrderProduct) (uuid.UUID, error) // тут dto или даже мапа
	RejectOrder(orderId uuid.UUID) error
	GetUnpaidProductCounts(ctx context.Context, userId uuid.UUID) ([]model.ProductCount, error)
	GetOrderById(ctx context.Context, orderId uuid.UUID) (*model.Order, error)
	MarkOrdersPaid(ctx context.Context, orderId uuid.UUID) error
}

type OrderRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func (o *OrderRepositoryImpl) GetOrderProducts(orderId uuid.UUID) ([]model.OrderProduct, error) {
	query := "SELECT po.product_id, po.product_count FROM product_order_count JOIN public.product_order po on po.order_id = product_order_count.product_order_fk WHERE order_id=$1"
	rows, err := o.db.Query(query, orderId)
	if err != nil {
		o.logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	var products []model.OrderProduct
	for rows.Next() {
		var product model.OrderProduct
		if err := rows.Scan(&product.ProductId, &product.Count); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		o.logger.Error(err.Error())
		return nil, err
	}
	return products, nil
}

func (o *OrderRepositoryImpl) GetOrderState(orderId uuid.UUID) (string, error) {
	query := "SELECT order_status FROM product_order where order_id=$1"
	var status string
	err := o.db.QueryRow(query, orderId).Scan(&status)
	if err != nil {
		o.logger.Error(err.Error())
		return "", err
	}
	return status, nil
}

func (o *OrderRepositoryImpl) GetUnpaidProductCounts(ctx context.Context, userId uuid.UUID) ([]model.ProductCount, error) {
	query := `
		SELECT poc.product_id, SUM(poc.product_count) AS total_count
		FROM product_order po
		JOIN product_order_count poc ON poc.product_order_fk = po.order_id
		WHERE po.user_id = $1 AND po.order_status = $2
		GROUP BY poc.product_id
	`
	rows, err := o.db.QueryContext(ctx, query, userId, orderv1.OrderState_WAIT_PAID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ProductCount
	for rows.Next() {
		var pc model.ProductCount
		if err := rows.Scan(&pc.ProductId, &pc.Count); err != nil {
			return nil, err
		}
		result = append(result, pc)
	}
	return result, rows.Err()
}

func (o *OrderRepositoryImpl) CreateOrder(products []model.OrderProduct) (uuid.UUID, error) {
	//TODO implement me
	//
	panic("implement me")
}

func (o *OrderRepositoryImpl) RejectOrder(orderId uuid.UUID) error {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepositoryImpl) GetOrderById(ctx context.Context, orderId uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := o.db.QueryRowContext(ctx, `
		SELECT order_id, user_id
		FROM product_order
		WHERE order_id = $1
	`, orderId).Scan(&order.OrderID, &order.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

func (o *OrderRepositoryImpl) MarkOrdersPaid(ctx context.Context, userId uuid.UUID) error {
	_, err := o.db.ExecContext(ctx, `
		UPDATE product_order
		SET order_status = $1
		WHERE user_id = $2 AND order_status = $3
	`, orderv1.OrderState_PAID.String(), userId, orderv1.OrderState_WAIT_PAID.String())
	return err
}

func NewOrderRepository(db *sql.DB, logger *slog.Logger) OrderRepository {
	return &OrderRepositoryImpl{db: db, logger: logger}
}
