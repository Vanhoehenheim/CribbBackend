// models/shopping_cart_activity.go
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityType defines the type of action performed on a shopping cart item
type CartActivityType string

const (
	// CartActivityTypeAdd indicates an item was added to the shopping cart
	CartActivityTypeAdd CartActivityType = "add"

	// CartActivityTypeUpdate indicates an item was updated
	CartActivityTypeUpdate CartActivityType = "update"

	// CartActivityTypeDelete indicates an item was removed from the shopping cart
	CartActivityTypeDelete CartActivityType = "delete"
)

// ShoppingCartActivity represents a record of changes to a shopping cart item
type ShoppingCartActivity struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	GroupID   primitive.ObjectID   `bson:"group_id" json:"group_id" validate:"required"`
	ItemID    primitive.ObjectID   `bson:"item_id" json:"item_id" validate:"required"`
	ItemName  string               `bson:"item_name" json:"item_name" validate:"required"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id" validate:"required"`
	UserName  string               `bson:"user_name" json:"user_name"`
	Action    CartActivityType     `bson:"action" json:"action" validate:"required"`
	Quantity  float64              `bson:"quantity" json:"quantity"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	Details   string               `bson:"details,omitempty" json:"details,omitempty"`
	IsRead    bool                 `bson:"is_read" json:"is_read"`
	ReadBy    []primitive.ObjectID `bson:"read_by" json:"read_by"`
	ExpiresAt time.Time            `bson:"expires_at" json:"expires_at"`
}

// CreateShoppingCartActivity creates a new shopping cart activity record
func CreateShoppingCartActivity(
	groupID primitive.ObjectID,
	itemID primitive.ObjectID,
	itemName string,
	userID primitive.ObjectID,
	userName string,
	action CartActivityType,
	quantity float64,
	details string,
) *ShoppingCartActivity {
	return &ShoppingCartActivity{
		GroupID:   groupID,
		ItemID:    itemID,
		ItemName:  itemName,
		UserID:    userID,
		UserName:  userName,
		Action:    action,
		Quantity:  quantity,
		CreatedAt: time.Now(),
		Details:   details,
		IsRead:    false,
		ReadBy:    []primitive.ObjectID{},
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Activities expire after 7 days
	}
}

// HasBeenReadBy checks if the activity has been read by a specific user
func (a *ShoppingCartActivity) HasBeenReadBy(userID primitive.ObjectID) bool {
	for _, id := range a.ReadBy {
		if id == userID {
			return true
		}
	}
	return false
}

// MarkAsReadBy marks the activity as read by a specific user
func (a *ShoppingCartActivity) MarkAsReadBy(userID primitive.ObjectID) {
	// Check if already read by this user
	if a.HasBeenReadBy(userID) {
		return
	}

	// Add user to the read_by list
	a.ReadBy = append(a.ReadBy, userID)

	// If all members of the group have read it, mark as read
	// This would require checking against all group members, which
	// we'll leave to the handler implementation
}
