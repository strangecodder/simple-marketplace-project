package model

import "github.com/google/uuid"

type OrderProduct struct {
	ProductId uuid.UUID `json:"product_id"`
	Count     int64
}

type CreateProductsDto struct {
}

type Order struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
}

type UserOrderTotal struct {
	UserID uuid.UUID
	Total  int64
}

type ProductCount struct {
	ProductId uuid.UUID
	Count     int64
}
