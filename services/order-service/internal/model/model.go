package model

import "github.com/google/uuid"

type OrderProduct struct {
	ProductId uuid.UUID `json:"product_id"`
	Count     int64
}

type CreateProductsDto struct {
}
