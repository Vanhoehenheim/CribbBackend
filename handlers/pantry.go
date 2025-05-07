package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddPantryItemRequest defines the request structure for adding a pantry item
type AddPantryItemRequest struct {
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	Category       string  `json:"category"`
	ExpirationDate *string `json:"expiration_date,omitempty"`
	GroupName      string  `json:"group_name"`
}

// UsePantryItemRequest defines the request structure for using a pantry item
type UsePantryItemRequest struct {
	ItemID   string  `json:"item_id"`
	Quantity float64 `json:"quantity"`
}

// UsePantryItemHandler handles consuming an item from the pantry
func UsePantryItemHandler(w http.ResponseWriter, r *http.Request) {
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

	var request UsePantryItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ItemID == "" || request.Quantity <= 0 {
		http.Error(w, "Item ID and quantity are required. Quantity must be positive.", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Convert item ID to ObjectID
	itemID, err := primitive.ObjectIDFromHex(request.ItemID)
	if err != nil {
		http.Error(w, "Invalid item ID format", http.StatusBadRequest)
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

	// Start a transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Failed to start MongoDB session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Response data structure
	type UsePantryItemResponse struct {
		Success      bool    `json:"success"`
		Message      string  `json:"message"`
		RemainingQty float64 `json:"remaining_quantity"`
		Unit         string  `json:"unit"`
	}
	var response UsePantryItemResponse

	// Start transaction
	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// Find the pantry item
		var pantryItem models.PantryItem
		err := config.DB.Collection("pantry_items").FindOne(
			sc,
			bson.M{"_id": itemID},
		).Decode(&pantryItem)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return errors.New("pantry item not found")
			}
			return err
		}

		// Verify the item belongs to the user's group
		if pantryItem.GroupID != user.GroupID {
			return errors.New("pantry item does not belong to user's group")
		}

		// Check if there's enough quantity
		if pantryItem.Quantity < request.Quantity {
			return errors.New("not enough quantity available")
		}

		// Update the quantity
		newQuantity := pantryItem.Quantity - request.Quantity
		pantryItem.UpdateQuantity(newQuantity)

		_, err = config.DB.Collection("pantry_items").UpdateOne(
			sc,
			bson.M{"_id": pantryItem.ID},
			bson.M{"$set": bson.M{
				"quantity":   newQuantity,
				"updated_at": pantryItem.UpdatedAt,
			}},
		)
		if err != nil {
			return err
		}

		// Set response values
		response.Success = true
		response.Message = "Item used successfully"
		response.RemainingQty = newQuantity
		response.Unit = pantryItem.Unit

		// Check if low-stock notification is needed (if quantity is below 20%)
		// This threshold could be made configurable in the future
		if newQuantity > 0 && newQuantity <= 1 {
			notification := models.CreatePantryNotification(
				pantryItem.GroupID,
				pantryItem.ID,
				pantryItem.Name,
				models.NotificationTypeLowStock,
				"Item is running low",
			)
			_, err = config.DB.Collection("pantry_notifications").InsertOne(sc, notification)
			if err != nil {
				log.Printf("Failed to create low-stock notification: %v", err)
				// Continue anyway, as this is not critical
			}
		}

		if newQuantity == 0 {
			// Remove any existing low_stock notifications
			_, err = config.DB.Collection("pantry_notifications").DeleteMany(
				sc,
				bson.M{
					"item_id": pantryItem.ID,
					"type":    models.NotificationTypeLowStock,
				},
			)
			if err != nil {
				log.Printf("Failed to delete low_stock notifications: %v", err)
				// Continue anyway as this is not critical
			}

			// Create out_of_stock notification
			notification := models.CreatePantryNotification(
				pantryItem.GroupID,
				pantryItem.ID,
				pantryItem.Name,
				models.NotificationTypeOutOfStock,
				"Item is out of stock",
			)

			_, err = config.DB.Collection("pantry_notifications").InsertOne(sc, notification)
			if err != nil {
				log.Printf("Failed to create out_of_stock notification: %v", err)
				// Continue anyway as this is not critical
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create history record for using an item
	itemID, _ = primitive.ObjectIDFromHex(request.ItemID)
	var pantryItem models.PantryItem
	err = config.DB.Collection("pantry_items").FindOne(
		context.Background(),
		bson.M{"_id": itemID},
	).Decode(&pantryItem)

	if err == nil {
		UpdatePantryHistoryForUse(
			user.GroupID,
			itemID,
			pantryItem.Name,
			userID,
			user.Name,
			request.Quantity,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func AddPantryItemHandler(w http.ResponseWriter, r *http.Request) {
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

	var request AddPantryItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.Name == "" || request.Quantity < 0 || request.Unit == "" || request.GroupName == "" {
		http.Error(w, "Name, quantity, unit, and group name are required", http.StatusBadRequest)
		return
	}

	// Find the group
	var group models.Group
	err := config.DB.Collection("groups").FindOne(
		context.Background(),
		bson.M{"name": request.GroupName},
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
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

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Parse expiration date if provided
	var expirationDate time.Time
	if request.ExpirationDate != nil && *request.ExpirationDate != "" {
		expirationDate, err = time.Parse(time.RFC3339, *request.ExpirationDate)
		if err != nil {
			http.Error(w, "Invalid expiration date format. Use ISO 8601/RFC3339 format (YYYY-MM-DDTHH:MM:SSZ)", http.StatusBadRequest)
			return
		}
	}

	// Start a transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Failed to start MongoDB session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Start transaction
	var pantryItem models.PantryItem
	var isNewItem bool = true
	var oldQuantity float64 = 0

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// Check if item already exists in this group
		existingItem := config.DB.Collection("pantry_items").FindOne(
			sc,
			bson.M{
				"group_id": group.ID,
				"name":     bson.M{"$regex": primitive.Regex{Pattern: "^" + strings.TrimSpace(request.Name) + "$", Options: "i"}},
			},
		)

		if existingItem.Err() == nil {
			// Item exists, update it
			if err := existingItem.Decode(&pantryItem); err != nil {
				return err
			}

			isNewItem = false
			oldQuantity = pantryItem.Quantity

			// Update the item
			pantryItem.Quantity = request.Quantity
			pantryItem.Unit = request.Unit
			if request.Category != "" {
				pantryItem.Category = request.Category
			}
			if !expirationDate.IsZero() {
				pantryItem.ExpirationDate = expirationDate
			}
			pantryItem.UpdatedAt = time.Now()

			_, err = config.DB.Collection("pantry_items").UpdateOne(
				sc,
				bson.M{"_id": pantryItem.ID},
				bson.M{"$set": pantryItem},
			)
			if err != nil {
				return err
			}
		} else if errors.Is(existingItem.Err(), mongo.ErrNoDocuments) {
			// Item doesn't exist, create new one
			category := request.Category
			if category == "" {
				category = "Uncategorized"
			}

			pantryItem = *models.CreatePantryItem(
				group.ID,
				request.Name,
				request.Quantity,
				request.Unit,
				category,
				expirationDate,
				userID,
			)

			result, err := config.DB.Collection("pantry_items").InsertOne(sc, pantryItem)
			if err != nil {
				return err
			}
			pantryItem.ID = result.InsertedID.(primitive.ObjectID)
		} else {
			// Some other error occurred
			return existingItem.Err()
		}

		// Check if we need to create expiration notification
		if !expirationDate.IsZero() && pantryItem.IsExpiringSoon(3) {
			notification := models.CreatePantryNotification(
				group.ID,
				pantryItem.ID,
				pantryItem.Name,
				models.NotificationTypeExpiringSoon,
				"Item will expire in 3 days or less",
			)
			_, err = config.DB.Collection("pantry_notifications").InsertOne(sc, notification)
			if err != nil {
				log.Printf("Failed to create expiration notification: %v", err)
				// Continue anyway, as this is not critical
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		http.Error(w, "Failed to add/update pantry item", http.StatusInternalServerError)
		return
	}

	// Create history record for adding a new item or updating an existing one
	if isNewItem {
		UpdatePantryHistoryForAdd(
			group.ID,
			pantryItem.ID,
			pantryItem.Name,
			userID,
			user.Name,
			request.Quantity,
		)
	} else {
		// If quantity changed, record it as an update
		if oldQuantity != request.Quantity {
			UpdatePantryHistoryForAdd(
				group.ID,
				pantryItem.ID,
				pantryItem.Name,
				userID,
				user.Name,
				request.Quantity-oldQuantity,
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pantryItem)
}

// GetPantryItemsHandler retrieves all pantry items for a group
func GetPantryItemsHandler(w http.ResponseWriter, r *http.Request) {
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
	category := r.URL.Query().Get("category")

	// Verify group name is provided
	if groupName == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Find the group
	var group models.Group
	err := config.DB.Collection("groups").FindOne(
		context.Background(),
		bson.M{"name": groupName},
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
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

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Build query filter
	filter := bson.M{"group_id": group.ID}
	if category != "" {
		filter["category"] = category
	}

	// Find pantry items
	opts := options.Find().SetSort(bson.D{
		{Key: "category", Value: 1},
		{Key: "name", Value: 1},
	})

	cursor, err := config.DB.Collection("pantry_items").Find(
		context.Background(),
		filter,
		opts,
	)
	if err != nil {
		http.Error(w, "Failed to fetch pantry items", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var pantryItems []models.PantryItem
	if err = cursor.All(context.Background(), &pantryItems); err != nil {
		http.Error(w, "Failed to decode pantry items", http.StatusInternalServerError)
		return
	}

	// Extend the response with more information about each item
	type PantryItemResponse struct {
		models.PantryItem
		IsExpiringSoon bool   `json:"is_expiring_soon"`
		IsExpired      bool   `json:"is_expired"`
		AddedByName    string `json:"added_by_name"`
	}

	response := make([]PantryItemResponse, 0, len(pantryItems))
	for _, item := range pantryItems {
		extendedItem := PantryItemResponse{
			PantryItem:     item,
			IsExpiringSoon: item.IsExpiringSoon(3),
			IsExpired:      item.IsExpired(),
			AddedByName:    "",
		}

		// Get the name of the user who added the item
		var addedByUser models.User
		err = config.DB.Collection("users").FindOne(
			context.Background(),
			bson.M{"_id": item.AddedBy},
		).Decode(&addedByUser)
		if err == nil {
			extendedItem.AddedByName = addedByUser.Name
		}

		response = append(response, extendedItem)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePantryItemHandler handles deleting a pantry item
func DeletePantryItemHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get item ID from URL path
	itemIDStr := strings.TrimPrefix(r.URL.Path, "/api/pantry/remove/")
	if itemIDStr == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Convert item ID to ObjectID
	itemID, err := primitive.ObjectIDFromHex(itemIDStr)
	if err != nil {
		http.Error(w, "Invalid item ID format", http.StatusBadRequest)
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

	// Get item details before deletion for history
	var pantryItem models.PantryItem
	err = config.DB.Collection("pantry_items").FindOne(
		context.Background(),
		bson.M{"_id": itemID},
	).Decode(&pantryItem)

	var itemName string
	var itemQuantity float64
	var groupID primitive.ObjectID

	if err == nil {
		itemName = pantryItem.Name
		itemQuantity = pantryItem.Quantity
		groupID = pantryItem.GroupID
	}

	// Start a transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Failed to start MongoDB session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Execute the transaction
	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// Find the pantry item first to verify it belongs to the user's group
		var pantryItem models.PantryItem
		err := config.DB.Collection("pantry_items").FindOne(
			sc,
			bson.M{"_id": itemID},
		).Decode(&pantryItem)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return errors.New("pantry item not found")
			}
			return err
		}

		// Verify the item belongs to the user's group
		if pantryItem.GroupID != user.GroupID {
			return errors.New("pantry item does not belong to user's group")
		}

		// Delete the pantry item
		_, err = config.DB.Collection("pantry_items").DeleteOne(
			sc,
			bson.M{"_id": itemID},
		)
		if err != nil {
			return err
		}

		// Delete any notifications related to this item
		_, err = config.DB.Collection("pantry_notifications").DeleteMany(
			sc,
			bson.M{"item_id": itemID},
		)
		if err != nil {
			log.Printf("Failed to delete related notifications: %v", err)
			// Continue anyway, as this is not critical
		}

		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create history record for removing an item
	if itemName != "" {
		UpdatePantryHistoryForRemove(
			groupID,
			itemID,
			itemName,
			userID,
			user.Name,
			itemQuantity,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Pantry item deleted successfully",
	})
}
