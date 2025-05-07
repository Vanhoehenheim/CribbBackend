// test/mocks_shopping_cart.go
package test

import (
	"cribb-backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Update the TestDB struct in test/mocks.go to add ShoppingCartItems field:
/*
type TestDB struct {
	Users               []models.User
	Groups              []models.Group
	Chores              []models.Chore
	RecurringChores     []models.RecurringChore
	ChoreCompletions    []models.ChoreCompletion
	PantryItems         []models.PantryItem
	PantryNotifications []models.PantryNotification
	PantryHistory       []models.PantryHistory
	ShoppingCartItems   []models.ShoppingCartItem  // Add this line
}
*/

// AddShoppingCartItem adds a shopping cart item to the test database
func (db *TestDB) AddShoppingCartItem(item models.ShoppingCartItem) {
	// Initialize the slice if it's nil
	if db.ShoppingCartItems == nil {
		db.ShoppingCartItems = make([]models.ShoppingCartItem, 0)
	}
	db.ShoppingCartItems = append(db.ShoppingCartItems, item)
}

// UpdateShoppingCartItem updates a shopping cart item in the test database
func (db *TestDB) UpdateShoppingCartItem(item models.ShoppingCartItem) {
	// Initialize the slice if it's nil
	if db.ShoppingCartItems == nil {
		db.ShoppingCartItems = make([]models.ShoppingCartItem, 0)
		db.AddShoppingCartItem(item)
		return
	}

	for i, existingItem := range db.ShoppingCartItems {
		if existingItem.ID == item.ID {
			db.ShoppingCartItems[i] = item
			return
		}
	}
	// If not found, add it
	db.AddShoppingCartItem(item)
}

// DeleteShoppingCartItem deletes a shopping cart item from the test database
func (db *TestDB) DeleteShoppingCartItem(id primitive.ObjectID) bool {
	if db.ShoppingCartItems == nil {
		return false
	}

	for i, item := range db.ShoppingCartItems {
		if item.ID == id {
			db.ShoppingCartItems = append(db.ShoppingCartItems[:i], db.ShoppingCartItems[i+1:]...)
			return true
		}
	}
	return false
}

// FindShoppingCartItemByID finds a shopping cart item by ID
func (db *TestDB) FindShoppingCartItemByID(id primitive.ObjectID) (models.ShoppingCartItem, bool) {
	if db.ShoppingCartItems == nil {
		return models.ShoppingCartItem{}, false
	}

	for _, item := range db.ShoppingCartItems {
		if item.ID == id {
			return item, true
		}
	}
	return models.ShoppingCartItem{}, false
}

// GetShoppingCartItemsByGroup gets all shopping cart items for a group
func (db *TestDB) GetShoppingCartItemsByGroup(groupID primitive.ObjectID) []models.ShoppingCartItem {
	var result []models.ShoppingCartItem

	if db.ShoppingCartItems == nil {
		return result
	}

	for _, item := range db.ShoppingCartItems {
		if item.GroupID == groupID {
			result = append(result, item)
		}
	}
	return result
}

// GetShoppingCartItemsByUser gets all shopping cart items for a user
func (db *TestDB) GetShoppingCartItemsByUser(userID primitive.ObjectID) []models.ShoppingCartItem {
	var result []models.ShoppingCartItem

	if db.ShoppingCartItems == nil {
		return result
	}

	for _, item := range db.ShoppingCartItems {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result
}

// GetShoppingCartItemsByGroupAndUser gets all shopping cart items for a specific group and user
func (db *TestDB) GetShoppingCartItemsByGroupAndUser(groupID, userID primitive.ObjectID) []models.ShoppingCartItem {
	var result []models.ShoppingCartItem

	if db.ShoppingCartItems == nil {
		return result
	}

	for _, item := range db.ShoppingCartItems {
		if item.GroupID == groupID && item.UserID == userID {
			result = append(result, item)
		}
	}
	return result
}
