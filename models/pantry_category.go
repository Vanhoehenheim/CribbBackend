package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PantryCategoryType defines the type of pantry category
type PantryCategoryType string

const (
	// CategoryTypePredefined indicates a system-wide predefined category
	CategoryTypePredefined PantryCategoryType = "predefined"

	// CategoryTypeCustom indicates a group-specific custom category
	CategoryTypeCustom PantryCategoryType = "custom"
)

// PantryCategory represents a category for organizing pantry items
type PantryCategory struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name      string              `bson:"name" json:"name" validate:"required,min=1"`
	Type      PantryCategoryType  `bson:"type" json:"type" validate:"required"`
	GroupID   *primitive.ObjectID `bson:"group_id,omitempty" json:"group_id,omitempty"`     // null for predefined, group_id for custom
	CreatedBy *primitive.ObjectID `bson:"created_by,omitempty" json:"created_by,omitempty"` // null for predefined, user_id for custom
	CreatedAt time.Time           `bson:"created_at" json:"created_at"`
	IsActive  bool                `bson:"is_active" json:"is_active"`
}

// CreatePredefinedCategory creates a new predefined category
func CreatePredefinedCategory(name string) *PantryCategory {
	return &PantryCategory{
		Name:      name,
		Type:      CategoryTypePredefined,
		GroupID:   nil,
		CreatedBy: nil,
		CreatedAt: time.Now(),
		IsActive:  true,
	}
}

// CreateCustomCategory creates a new custom category for a group
func CreateCustomCategory(name string, groupID primitive.ObjectID, createdBy primitive.ObjectID) *PantryCategory {
	return &PantryCategory{
		Name:      name,
		Type:      CategoryTypeCustom,
		GroupID:   &groupID,
		CreatedBy: &createdBy,
		CreatedAt: time.Now(),
		IsActive:  true,
	}
}

// IsPredefined checks if the category is a predefined category
func (pc *PantryCategory) IsPredefined() bool {
	return pc.Type == CategoryTypePredefined
}

// IsCustom checks if the category is a custom category
func (pc *PantryCategory) IsCustom() bool {
	return pc.Type == CategoryTypeCustom
}

// BelongsToGroup checks if the category belongs to a specific group
func (pc *PantryCategory) BelongsToGroup(groupID primitive.ObjectID) bool {
	if pc.IsPredefined() {
		return true // Predefined categories belong to all groups
	}
	return pc.GroupID != nil && *pc.GroupID == groupID
}

// CanBeEditedBy checks if a user can edit this category
func (pc *PantryCategory) CanBeEditedBy(userID primitive.ObjectID, userGroupID primitive.ObjectID) bool {
	if pc.IsPredefined() {
		return false // Predefined categories cannot be edited by users
	}
	// Custom categories can be edited by members of the same group
	return pc.GroupID != nil && *pc.GroupID == userGroupID
}

// CanBeDeletedBy checks if a user can delete this category
func (pc *PantryCategory) CanBeDeletedBy(userID primitive.ObjectID, userGroupID primitive.ObjectID) bool {
	if pc.IsPredefined() {
		return false // Predefined categories cannot be deleted by users
	}
	// Custom categories can be deleted by members of the same group
	return pc.GroupID != nil && *pc.GroupID == userGroupID
}
