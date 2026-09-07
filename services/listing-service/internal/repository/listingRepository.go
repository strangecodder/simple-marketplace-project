package repository

import (
	"context"
	"database/sql"
	"errors"
	"listing-service/internal/model"
	"log/slog"

	"github.com/google/uuid"
)

type Repository interface {
	FindAllProducts(ctx context.Context) ([]model.Product, error)
	FindProductById(ctx context.Context, productId uuid.UUID) (model.Product, error)
	CreateProduct(ctx context.Context, sellerId uuid.UUID, name string, description string, price float64) (uuid.UUID, error)
	UpdateProduct(ctx context.Context, productId uuid.UUID, sellerId uuid.UUID, name string, description string, price int64) error
	DeleteProduct(ctx context.Context, productId uuid.UUID) error
}

type RepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func (r RepositoryImpl) FindAllProducts(ctx context.Context) ([]model.Product, error) {
	//TODO implement me
	r.db.QueryRow("SELECT * FROM product")
	return nil, nil
}

func (r RepositoryImpl) FindProductById(ctx context.Context, productId uuid.UUID) (model.Product, error) {
	var product model.Product
	err := r.db.QueryRow("SELECT * FROM product WHERE product_id = $1", productId).Scan(&product)
	if err != nil {
		r.logger.Error(err.Error())
		return model.Product{}, err
	}
	return product, nil
}

func (r RepositoryImpl) CreateProduct(ctx context.Context, sellerId uuid.UUID, name string, description string, price float64) (uuid.UUID, error) {
	var productId uuid.UUID
	query := "INSERT INTO product(seller_id, product_name, product_description, price) VALUES ($1,$2,$3,$4) RETURNING product_id;"
	err := r.db.QueryRow(query, sellerId, name, description, price).Scan(&productId)
	if err != nil {
		r.logger.Error(err.Error())
		return uuid.Nil, err
	}
	return productId, nil
}

func (r RepositoryImpl) UpdateProduct(ctx context.Context, productId uuid.UUID, sellerId uuid.UUID, name string, description string, price int64) error {
	_, err := r.FindProductById(ctx, productId)
	if err != nil {
		r.logger.Error(err.Error())
		return err
	}
	return nil
}

func (r RepositoryImpl) DeleteProduct(ctx context.Context, productId uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.Error(err.Error())
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.logger.Error("rollback failed: %s", err.Error())
		}
	}()

	deleteActionQuery := "DELETE FROM product_action WHERE product_id_fk = $1"
	if _, err = tx.ExecContext(ctx, deleteActionQuery, productId); err != nil {
		r.logger.Error(err.Error())
		return err
	}

	query := "DELETE FROM product WHERE product_id = $1"
	if _, err = tx.ExecContext(ctx, query, productId); err != nil {
		r.logger.Error(err.Error())
		return err
	}
	if err = tx.Commit(); err != nil {
		r.logger.Error(err.Error())
		return err
	}
	return nil
}

func NewRepository(db *sql.DB, logger *slog.Logger) Repository {
	return RepositoryImpl{db: db, logger: logger}
}
