// models/pantry_history.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActionType defines the type of action performed on a pantry item
type ActionType string

const (
	// ActionTypeAdd indicates an item was added to the pantry
	ActionTypeAdd ActionType = "add"

	// ActionTypeUpdate indicates an item was updated
	ActionTypeUpdate ActionType = "update"

	// ActionTypeUse indicates an item was used/consumed
	ActionTypeUse ActionType = "use"

	// ActionTypeRemove indicates an item was removed from the pantry
	ActionTypeRemove ActionType = "remove"
)

// PantryHistory represents a record of changes to a pantry item
type PantryHistory struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GroupID   primitive.ObjectID `bson:"group_id" json:"group_id" validate:"required"`
	ItemID    primitive.ObjectID `bson:"item_id" json:"item_id" validate:"required"`
	ItemName  string             `bson:"item_name" json:"item_name" validate:"required"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id" validate:"required"`
	UserName  string             `bson:"user_name" json:"user_name"`
	Action    ActionType         `bson:"action" json:"action" validate:"required"`
	Quantity  float64            `bson:"quantity" json:"quantity"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	Details   string             `bson:"details,omitempty" json:"details,omitempty"`
}

// CreatePantryHistory creates a new pantry history record
func CreatePantryHistory(
	groupID primitive.ObjectID,
	itemID primitive.ObjectID,
	itemName string,
	userID primitive.ObjectID,
	userName string,
	action ActionType,
	quantity float64,
	details string,
) *PantryHistory {
	return &PantryHistory{
		GroupID:   groupID,
		ItemID:    itemID,
		ItemName:  itemName,
		UserID:    userID,
		UserName:  userName,
		Action:    action,
		Quantity:  quantity,
		CreatedAt: time.Now(),
		Details:   details,
	}
}
