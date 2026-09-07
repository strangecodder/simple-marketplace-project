package repository

import (
	"database/sql"
	"log/slog"
	"order-service/internal/model"

	"github.com/google/uuid"
)

type OrderRepository interface {
	GetOrderProducts(orderId uuid.UUID) ([]model.OrderProduct, error)
	GetOrderState(orderId uuid.UUID) (string, error)
	CreateOrder(products []model.OrderProduct) (uuid.UUID, error) // тут dto или даже мапа
	RejectOrder(orderId uuid.UUID) error
}

type OrderRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func (o OrderRepositoryImpl) GetOrderProducts(orderId uuid.UUID) ([]model.OrderProduct, error) {
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

func (o OrderRepositoryImpl) GetOrderState(orderId uuid.UUID) (string, error) {
	query := "SELECT order_status FROM product_order where order_id=$1"
	var status string
	err := o.db.QueryRow(query, orderId).Scan(&status)
	if err != nil {
		o.logger.Error(err.Error())
		return "", err
	}
	return status, nil
}

func (o OrderRepositoryImpl) CreateOrder(products []model.OrderProduct) (uuid.UUID, error) {
	//TODO implement me
	panic("implement me")
}

func (o OrderRepositoryImpl) RejectOrder(orderId uuid.UUID) error {
	//TODO implement me
	panic("implement me")
}

func NewOrderRepository(db *sql.DB, logger *slog.Logger) OrderRepository {
	return &OrderRepositoryImpl{db: db, logger: logger}
}
