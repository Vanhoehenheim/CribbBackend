package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ShoppingCartItem represents an item in a user's shopping cart
type ShoppingCartItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	GroupID  primitive.ObjectID `bson:"group_id" json:"group_id" validate:"required"`
	ItemName string             `bson:"item_name" json:"item_name" validate:"required"`
	Quantity float64            `bson:"quantity" json:"quantity" validate:"required,min=0.1"`
	Category string             `bson:"category" json:"category"`
	AddedAt  time.Time          `bson:"added_at" json:"added_at"`
}

// CreateShoppingCartItem creates a new shopping cart item
func CreateShoppingCartItem(
	userID primitive.ObjectID,
	groupID primitive.ObjectID,
	itemName string,
	quantity float64,
	category string,
) *ShoppingCartItem {
	return &ShoppingCartItem{
		UserID:   userID,
		GroupID:  groupID,
		ItemName: itemName,
		Quantity: quantity,
		Category: category,
		AddedAt:  time.Now(),
	}
}

// UpdateQuantity updates the item's quantity
func (s *ShoppingCartItem) UpdateQuantity(newQuantity float64) {
	s.Quantity = newQuantity
}
