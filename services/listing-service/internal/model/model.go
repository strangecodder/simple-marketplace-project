package model

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ProductId   uuid.UUID `json:"product_id"`
	SellerId    uuid.UUID `json:"seller_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Count       int64     `json:"count"`
}

type ProductAction struct {
	ProductId   uuid.UUID `json:"product_id"`
	Action      string    `json:"action"`
	ActionValue string    `json:"value"`
	Time        time.Time `json:"time"`
}
