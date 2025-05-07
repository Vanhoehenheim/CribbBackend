// test/mocks_pantry.go
package test

import (
	"cribb-backend/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Instead of overriding NewTestDB, we'll just add our pantry-related functions
// to the TestDB struct, assuming the TestDB is already properly initialized

// AddPantryItem adds a pantry item to the test database
func (db *TestDB) AddPantryItem(item models.PantryItem) {
	// Initialize the slice if it's nil
	if db.PantryItems == nil {
		db.PantryItems = make([]models.PantryItem, 0)
	}
	db.PantryItems = append(db.PantryItems, item)
}

// UpdatePantryItem updates a pantry item in the test database
func (db *TestDB) UpdatePantryItem(item models.PantryItem) {
	// Initialize the slice if it's nil
	if db.PantryItems == nil {
		db.PantryItems = make([]models.PantryItem, 0)
		db.AddPantryItem(item)
		return
	}

	for i, existingItem := range db.PantryItems {
		if existingItem.ID == item.ID {
			db.PantryItems[i] = item
			return
		}
	}
	// If not found, add it
	db.AddPantryItem(item)
}

// DeletePantryItem deletes a pantry item from the test database
func (db *TestDB) DeletePantryItem(id primitive.ObjectID) bool {
	if db.PantryItems == nil {
		return false
	}

	for i, item := range db.PantryItems {
		if item.ID == id {
			db.PantryItems = append(db.PantryItems[:i], db.PantryItems[i+1:]...)
			return true
		}
	}
	return false
}

// FindPantryItemByID finds a pantry item by ID
func (db *TestDB) FindPantryItemByID(id primitive.ObjectID) (models.PantryItem, bool) {
	if db.PantryItems == nil {
		return models.PantryItem{}, false
	}

	for _, item := range db.PantryItems {
		if item.ID == id {
			return item, true
		}
	}
	return models.PantryItem{}, false
}

// GetPantryItemsByGroup gets all pantry items for a group
func (db *TestDB) GetPantryItemsByGroup(groupID primitive.ObjectID) []models.PantryItem {
	var result []models.PantryItem

	if db.PantryItems == nil {
		return result
	}

	for _, item := range db.PantryItems {
		if item.GroupID == groupID {
			result = append(result, item)
		}
	}
	return result
}

// GetPantryItemsByCategory gets all pantry items for a group in a specific category
func (db *TestDB) GetPantryItemsByCategory(groupID primitive.ObjectID, category string) []models.PantryItem {
	var result []models.PantryItem

	if db.PantryItems == nil {
		return result
	}

	for _, item := range db.PantryItems {
		if item.GroupID == groupID && item.Category == category {
			result = append(result, item)
		}
	}
	return result
}

// AddPantryNotification adds a pantry notification to the test database
func (db *TestDB) AddPantryNotification(notification models.PantryNotification) {
	// Initialize the slice if it's nil
	if db.PantryNotifications == nil {
		db.PantryNotifications = make([]models.PantryNotification, 0)
	}
	db.PantryNotifications = append(db.PantryNotifications, notification)
}

// UpdatePantryNotification updates a pantry notification in the test database
func (db *TestDB) UpdatePantryNotification(notification models.PantryNotification) {
	// Initialize the slice if it's nil
	if db.PantryNotifications == nil {
		db.PantryNotifications = make([]models.PantryNotification, 0)
		db.AddPantryNotification(notification)
		return
	}

	for i, existingNotification := range db.PantryNotifications {
		if existingNotification.ID == notification.ID {
			db.PantryNotifications[i] = notification
			return
		}
	}
	// If not found, add it
	db.AddPantryNotification(notification)
}

// DeletePantryNotification deletes a pantry notification from the test database
func (db *TestDB) DeletePantryNotification(id primitive.ObjectID) bool {
	if db.PantryNotifications == nil {
		return false
	}

	for i, notification := range db.PantryNotifications {
		if notification.ID == id {
			db.PantryNotifications = append(db.PantryNotifications[:i], db.PantryNotifications[i+1:]...)
			return true
		}
	}
	return false
}

// FindPantryNotificationByID finds a pantry notification by ID
func (db *TestDB) FindPantryNotificationByID(id primitive.ObjectID) (models.PantryNotification, bool) {
	if db.PantryNotifications == nil {
		return models.PantryNotification{}, false
	}

	for _, notification := range db.PantryNotifications {
		if notification.ID == id {
			return notification, true
		}
	}
	return models.PantryNotification{}, false
}

// GetPantryNotificationsByGroup gets all pantry notifications for a group
func (db *TestDB) GetPantryNotificationsByGroup(groupID primitive.ObjectID) []models.PantryNotification {
	var result []models.PantryNotification

	if db.PantryNotifications == nil {
		return result
	}

	for _, notification := range db.PantryNotifications {
		if notification.GroupID == groupID {
			result = append(result, notification)
		}
	}
	return result
}

// GetPantryNotificationsByType gets all pantry notifications for a group of a specific type
func (db *TestDB) GetPantryNotificationsByType(groupID primitive.ObjectID, notificationType models.NotificationType) []models.PantryNotification {
	var result []models.PantryNotification

	if db.PantryNotifications == nil {
		return result
	}

	for _, notification := range db.PantryNotifications {
		if notification.GroupID == groupID && notification.Type == notificationType {
			result = append(result, notification)
		}
	}
	return result
}

// AddPantryHistory adds a pantry history record to the test database
func (db *TestDB) AddPantryHistory(history models.PantryHistory) {
	// Initialize the slice if it's nil
	if db.PantryHistory == nil {
		db.PantryHistory = make([]models.PantryHistory, 0)
	}
	db.PantryHistory = append(db.PantryHistory, history)
}

// GetPantryHistoryByItemID gets all pantry history records for an item
func (db *TestDB) GetPantryHistoryByItemID(itemID primitive.ObjectID) []models.PantryHistory {
	var result []models.PantryHistory

	if db.PantryHistory == nil {
		return result
	}

	for _, history := range db.PantryHistory {
		if history.ItemID == itemID {
			result = append(result, history)
		}
	}
	return result
}

// GetPantryHistoryByGroup gets all pantry history records for a group
func (db *TestDB) GetPantryHistoryByGroup(groupID primitive.ObjectID) []models.PantryHistory {
	var result []models.PantryHistory

	if db.PantryHistory == nil {
		return result
	}

	for _, history := range db.PantryHistory {
		if history.GroupID == groupID {
			result = append(result, history)
		}
	}
	return result
}

// GetPantryHistoryByUser gets all pantry history records for a user
func (db *TestDB) GetPantryHistoryByUser(userID primitive.ObjectID) []models.PantryHistory {
	var result []models.PantryHistory

	if db.PantryHistory == nil {
		return result
	}

	for _, history := range db.PantryHistory {
		if history.UserID == userID {
			result = append(result, history)
		}
	}
	return result
}
