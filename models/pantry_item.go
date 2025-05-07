package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PantryItem represents an item in a group's shared pantry
type PantryItem struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GroupID        primitive.ObjectID `bson:"group_id" json:"group_id" validate:"required"`
	Name           string             `bson:"name" json:"name" validate:"required"`
	Quantity       float64            `bson:"quantity" json:"quantity" validate:"required,min=0"`
	Unit           string             `bson:"unit" json:"unit" validate:"required"`
	Category       string             `bson:"category" json:"category"`
	ExpirationDate time.Time          `bson:"expiration_date,omitempty" json:"expiration_date,omitempty"`
	AddedBy        primitive.ObjectID `bson:"added_by" json:"added_by" validate:"required"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// CreatePantryItem creates a new pantry item
func CreatePantryItem(
	groupID primitive.ObjectID,
	name string,
	quantity float64,
	unit string,
	category string,
	expirationDate time.Time,
	addedBy primitive.ObjectID,
) *PantryItem {
	return &PantryItem{
		GroupID:        groupID,
		Name:           name,
		Quantity:       quantity,
		Unit:           unit,
		Category:       category,
		ExpirationDate: expirationDate,
		AddedBy:        addedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// IsExpiringSoon checks if the item is expiring within the given number of days
func (p *PantryItem) IsExpiringSoon(days int) bool {
	if p.ExpirationDate.IsZero() {
		return false
	}

	expirationThreshold := time.Now().AddDate(0, 0, days)
	return p.ExpirationDate.Before(expirationThreshold) && p.ExpirationDate.After(time.Now())
}

// IsExpired checks if the item is already expired
func (p *PantryItem) IsExpired() bool {
	if p.ExpirationDate.IsZero() {
		return false
	}

	return p.ExpirationDate.Before(time.Now())
}

// UpdateQuantity updates the item's quantity and updated_at timestamp
func (p *PantryItem) UpdateQuantity(newQuantity float64) {
	p.Quantity = newQuantity
	p.UpdatedAt = time.Now()
}
