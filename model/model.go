package model

import "github.com/google/uuid"

// OrderRequest represents the input to the order workflow
type OrderRequest struct {
	UserID          uuid.UUID `json:"userID"`
	ProductID       uuid.UUID `json:"productid"`
	ProductQuantity int       `json:"productQuantity"`
}
