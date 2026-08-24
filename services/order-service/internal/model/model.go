package model

import "github.com/google/uuid"

type OrderProduct struct {
	ProductId uuid.UUID
	Count     int32
}
