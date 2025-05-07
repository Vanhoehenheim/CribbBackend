// handlers/pantry_management.go
package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/jobs"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetPantryWarningsHandler retrieves low-stock warnings for a group
// GetPantryWarningsHandler retrieves low-stock warnings for a group
func GetPantryWarningsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	// Need either group name or group code
	if groupName == "" && groupCode == "" {
		http.Error(w, "Group name or group code is required", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the group
	var groupFilter bson.M
	if groupName != "" {
		groupFilter = bson.M{"name": groupName}
	} else {
		groupFilter = bson.M{"group_code": groupCode}
	}

	var group models.Group
	err = config.DB.Collection("groups").FindOne(
		context.Background(),
		groupFilter,
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Find all low-stock and out-of-stock notifications for this group
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50)
	cursor, err := config.DB.Collection("pantry_notifications").Find(
		context.Background(),
		bson.M{
			"group_id": group.ID,
			"type": bson.M{
				"$in": []models.NotificationType{
					models.NotificationTypeLowStock,
					models.NotificationTypeOutOfStock,
				},
			},
		},
		opts,
	)

	if err != nil {
		http.Error(w, "Failed to fetch pantry warnings", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var notifications []models.PantryNotification
	if err = cursor.All(context.Background(), &notifications); err != nil {
		http.Error(w, "Failed to decode pantry warnings", http.StatusInternalServerError)
		return
	}

	// Now fetch the items to get current quantities
	type WarningResponse struct {
		models.PantryNotification
		CurrentQuantity float64 `json:"current_quantity"`
		Unit            string  `json:"unit"`
		IsRead          bool    `json:"is_read"`
	}

	response := make([]WarningResponse, 0, len(notifications))
	for _, notification := range notifications {
		warningResponse := WarningResponse{
			PantryNotification: notification,
			CurrentQuantity:    0,
			Unit:               "",
			IsRead:             notification.HasBeenReadBy(userID),
		}

		// Try to get the current item information
		var item models.PantryItem
		err := config.DB.Collection("pantry_items").FindOne(
			context.Background(),
			bson.M{"_id": notification.ItemID},
		).Decode(&item)

		if err == nil {
			warningResponse.CurrentQuantity = item.Quantity
			warningResponse.Unit = item.Unit
		}

		response = append(response, warningResponse)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPantryExpiringHandler retrieves items that are expiring soon
func GetPantryExpiringHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	// Need either group name or group code
	if groupName == "" && groupCode == "" {
		http.Error(w, "Group name or group code is required", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the group
	var groupFilter bson.M
	if groupName != "" {
		groupFilter = bson.M{"name": groupName}
	} else {
		groupFilter = bson.M{"group_code": groupCode}
	}

	var group models.Group
	err = config.DB.Collection("groups").FindOne(
		context.Background(),
		groupFilter,
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Find all expiration notifications for this group
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50)

	cursor, err := config.DB.Collection("pantry_notifications").Find(
		context.Background(),
		bson.M{
			"group_id": group.ID,
			"type": bson.M{
				"$in": []models.NotificationType{
					models.NotificationTypeExpiringSoon,
					models.NotificationTypeExpired,
				},
			},
		},
		opts,
	)

	if err != nil {
		http.Error(w, "Failed to fetch expiration notifications", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var notifications []models.PantryNotification
	if err = cursor.All(context.Background(), &notifications); err != nil {
		http.Error(w, "Failed to decode expiration notifications", http.StatusInternalServerError)
		return
	}

	// Now fetch the items to get current expiration dates
	type ExpiringResponse struct {
		models.PantryNotification
		ExpirationDate time.Time `json:"expiration_date"`
		Quantity       float64   `json:"quantity"`
		Unit           string    `json:"unit"`
		IsExpired      bool      `json:"is_expired"`
		IsRead         bool      `json:"is_read"`
	}

	response := make([]ExpiringResponse, 0, len(notifications))
	for _, notification := range notifications {
		expiringResponse := ExpiringResponse{
			PantryNotification: notification,
			IsRead:             notification.HasBeenReadBy(userID),
		}

		// Try to get the current item information
		var item models.PantryItem
		err := config.DB.Collection("pantry_items").FindOne(
			context.Background(),
			bson.M{"_id": notification.ItemID},
		).Decode(&item)

		if err == nil {
			expiringResponse.ExpirationDate = item.ExpirationDate
			expiringResponse.Quantity = item.Quantity
			expiringResponse.Unit = item.Unit
			expiringResponse.IsExpired = item.IsExpired()
		}

		response = append(response, expiringResponse)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// MarkNotificationReadHandler marks a notification as read by the current user
func MarkNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var request struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.NotificationID == "" {
		http.Error(w, "Notification ID is required", http.StatusBadRequest)
		return
	}

	// Convert notification ID to ObjectID
	notificationID, err := primitive.ObjectIDFromHex(request.NotificationID)
	if err != nil {
		http.Error(w, "Invalid notification ID format", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find notification
	var notification models.PantryNotification
	err = config.DB.Collection("pantry_notifications").FindOne(
		context.Background(),
		bson.M{"_id": notificationID},
	).Decode(&notification)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Notification not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch notification", http.StatusInternalServerError)
		}
		return
	}

	// Find user to verify group membership
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the notification's group
	if user.GroupID != notification.GroupID {
		http.Error(w, "User is not a member of this notification's group", http.StatusForbidden)
		return
	}

	// Mark notification as read
	_, err = config.DB.Collection("pantry_notifications").UpdateOne(
		context.Background(),
		bson.M{"_id": notificationID},
		bson.M{"$addToSet": bson.M{"read_by": userID}},
	)

	if err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Notification marked as read",
	})
}

// GetPantryShoppingListHandler generates a shopping list based on low stock items
func GetPantryShoppingListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	// Need either group name or group code
	if groupName == "" && groupCode == "" {
		http.Error(w, "Group name or group code is required", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the group
	var groupFilter bson.M
	if groupName != "" {
		groupFilter = bson.M{"name": groupName}
	} else {
		groupFilter = bson.M{"group_code": groupCode}
	}

	var group models.Group
	err = config.DB.Collection("groups").FindOne(
		context.Background(),
		groupFilter,
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Generate shopping list using the helper function in jobs package
	shoppingList, err := jobs.GenerateShoppingList(group.ID)
	if err != nil {
		http.Error(w, "Failed to generate shopping list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shopping_list": shoppingList,
	})
}

// GetPantryHistoryHandler retrieves the history of pantry actions
func GetPantryHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")
	itemID := r.URL.Query().Get("item_id")
	limit := 50 // Default limit

	// Need either group name or group code
	if groupName == "" && groupCode == "" {
		http.Error(w, "Group name or group code is required", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the group
	var groupFilter bson.M
	if groupName != "" {
		groupFilter = bson.M{"name": groupName}
	} else {
		groupFilter = bson.M{"group_code": groupCode}
	}

	var group models.Group
	err = config.DB.Collection("groups").FindOne(
		context.Background(),
		groupFilter,
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Set up the query filter
	filter := bson.M{"group_id": group.ID}

	// Add item ID filter if provided
	if itemID != "" {
		itemObjID, err := primitive.ObjectIDFromHex(itemID)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}
		filter["item_id"] = itemObjID
	}

	// Set up options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}). // Sort by newest first
		SetLimit(int64(limit))

	// Query the history collection
	cursor, err := config.DB.Collection("pantry_history").Find(
		context.Background(),
		filter,
		opts,
	)

	if err != nil {
		http.Error(w, "Failed to fetch pantry history", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var history []models.PantryHistory
	if err = cursor.All(context.Background(), &history); err != nil {
		http.Error(w, "Failed to decode pantry history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// DeleteNotificationHandler handles deleting a pantry notification
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get notification ID from query parameter
	notificationIDStr := r.URL.Query().Get("notification_id")
	if notificationIDStr == "" {
		http.Error(w, "Notification ID is required", http.StatusBadRequest)
		return
	}

	// Convert notification ID to ObjectID
	notificationID, err := primitive.ObjectIDFromHex(notificationIDStr)
	if err != nil {
		http.Error(w, "Invalid notification ID format", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find notification to verify ownership
	var notification models.PantryNotification
	err = config.DB.Collection("pantry_notifications").FindOne(
		context.Background(),
		bson.M{"_id": notificationID},
	).Decode(&notification)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Notification not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch notification", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the notification's group
	if user.GroupID != notification.GroupID {
		http.Error(w, "User is not a member of this notification's group", http.StatusForbidden)
		return
	}

	// Delete the notification
	result, err := config.DB.Collection("pantry_notifications").DeleteOne(
		context.Background(),
		bson.M{"_id": notificationID},
	)

	if err != nil {
		http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Notification not found or already deleted", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Notification deleted successfully",
	})
}

// UpdatePantryHistoryForAdd creates a history record for adding an item
func UpdatePantryHistoryForAdd(groupID, itemID primitive.ObjectID, itemName string, userID primitive.ObjectID, userName string, quantity float64) {
	history := models.CreatePantryHistory(
		groupID,
		itemID,
		itemName,
		userID,
		userName,
		models.ActionTypeAdd,
		quantity,
		"Item added to pantry",
	)

	_, err := config.DB.Collection("pantry_history").InsertOne(
		context.Background(),
		history,
	)

	if err != nil {
		log.Printf("Failed to create pantry history record: %v", err)
	}
}

// UpdatePantryHistoryForUse creates a history record for using an item
func UpdatePantryHistoryForUse(groupID, itemID primitive.ObjectID, itemName string, userID primitive.ObjectID, userName string, quantity float64) {
	history := models.CreatePantryHistory(
		groupID,
		itemID,
		itemName,
		userID,
		userName,
		models.ActionTypeUse,
		quantity,
		"Item used from pantry",
	)

	_, err := config.DB.Collection("pantry_history").InsertOne(
		context.Background(),
		history,
	)

	if err != nil {
		log.Printf("Failed to create pantry history record: %v", err)
	}
}

// UpdatePantryHistoryForRemove creates a history record for removing an item
func UpdatePantryHistoryForRemove(groupID, itemID primitive.ObjectID, itemName string, userID primitive.ObjectID, userName string, quantity float64) {
	history := models.CreatePantryHistory(
		groupID,
		itemID,
		itemName,
		userID,
		userName,
		models.ActionTypeRemove,
		quantity,
		"Item removed from pantry",
	)

	_, err := config.DB.Collection("pantry_history").InsertOne(
		context.Background(),
		history,
	)

	if err != nil {
		log.Printf("Failed to create pantry history record: %v", err)
	}
}
