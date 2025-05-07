package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationType defines the type of pantry notification
type NotificationType string

const (
	// NotificationTypeLowStock indicates an item is running low
	NotificationTypeLowStock NotificationType = "low_stock"

	// NotificationTypeExpiringSoon indicates an item is expiring soon
	NotificationTypeExpiringSoon NotificationType = "expiring_soon"

	// NotificationTypeExpired indicates an item has expired
	NotificationTypeExpired NotificationType = "expired"

	NotificationTypeOutOfStock NotificationType = "out_of_stock"
)

// PantryNotification represents a notification about a pantry item
type PantryNotification struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	GroupID   primitive.ObjectID   `bson:"group_id" json:"group_id" validate:"required"`
	ItemID    primitive.ObjectID   `bson:"item_id" json:"item_id" validate:"required"`
	ItemName  string               `bson:"item_name" json:"item_name" validate:"required"`
	Type      NotificationType     `bson:"type" json:"type" validate:"required"`
	Message   string               `bson:"message" json:"message"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	ReadBy    []primitive.ObjectID `bson:"read_by" json:"read_by"`
}

// CreatePantryNotification creates a new pantry notification
func CreatePantryNotification(
	groupID primitive.ObjectID,
	itemID primitive.ObjectID,
	itemName string,
	notificationType NotificationType,
	message string,
) *PantryNotification {
	return &PantryNotification{
		GroupID:   groupID,
		ItemID:    itemID,
		ItemName:  itemName,
		Type:      notificationType,
		Message:   message,
		CreatedAt: time.Now(),
		ReadBy:    make([]primitive.ObjectID, 0),
	}
}

// MarkAsReadByUser marks the notification as read by a specific user
func (n *PantryNotification) MarkAsReadByUser(userID primitive.ObjectID) {
	// Check if already marked as read by this user
	for _, id := range n.ReadBy {
		if id == userID {
			return
		}
	}

	// Add user to read_by list
	n.ReadBy = append(n.ReadBy, userID)
}

// HasBeenReadBy checks if the notification has been read by a specific user
func (n *PantryNotification) HasBeenReadBy(userID primitive.ObjectID) bool {
	for _, id := range n.ReadBy {
		if id == userID {
			return true
		}
	}
	return false
}
