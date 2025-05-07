// handlers/shopping_cart.go
package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddShoppingCartItemRequest defines the request structure for adding a shopping cart item
type AddShoppingCartItemRequest struct {
	ItemName string  `json:"item_name" validate:"required,min=1"`
	Quantity float64 `json:"quantity" validate:"required,min=0.1"`
	Category string  `json:"category"`
}

// UpdateShoppingCartItemRequest defines the request structure for updating a shopping cart item
type UpdateShoppingCartItemRequest struct {
	ItemID   string  `json:"item_id" validate:"required"`
	ItemName string  `json:"item_name,omitempty" validate:"min=1"`
	Quantity float64 `json:"quantity,omitempty" validate:"min=0.1"`
	Category string  `json:"category,omitempty"`
}

// Response structures
type ShoppingCartResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// AddShoppingCartItemHandler handles adding an item to the shopping cart
func AddShoppingCartItemHandler(w http.ResponseWriter, r *http.Request) {
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

	var request AddShoppingCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ItemName == "" || request.Quantity <= 0 {
		http.Error(w, "Item name and quantity are required. Quantity must be positive.", http.StatusBadRequest)
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

	// Define filter to find the item
	filter := bson.M{
		"user_id":   userID,
		"group_id":  user.GroupID,
		"item_name": request.ItemName,
	}

	// Variable to hold the final item state
	var finalShoppingCartItem models.ShoppingCartItem
	itemWasUpdated := false // Flag to track if we updated or inserted

	// Attempt to find the existing item first
	var existingItem models.ShoppingCartItem
	err = config.DB.Collection("shopping_cart").FindOne(context.Background(), filter).Decode(&existingItem)

	if err == nil {
		// Item found - Increment quantity and update timestamp/category
		itemWasUpdated = true
		update := bson.M{
			"$inc": bson.M{"quantity": request.Quantity}, // Increment quantity
			"$set": bson.M{
				"added_at": time.Now(), // Update timestamp
				// Optionally update category if provided?
				// "category": request.Category, // Keep existing category or update? Decide policy.
			},
		}
		// If category is provided in the request, update it as well
		if request.Category != "" {
			update["$set"].(bson.M)["category"] = request.Category
		}

		_, updateErr := config.DB.Collection("shopping_cart").UpdateOne(
			context.Background(),
			filter,
			update,
		)
		if updateErr != nil {
			log.Printf("Failed to increment shopping cart item quantity: %v", updateErr)
			http.Error(w, "Failed to update item quantity in shopping cart", http.StatusInternalServerError)
			return
		}
		// Fetch the updated item to return it
		fetchErr := config.DB.Collection("shopping_cart").FindOne(context.Background(), filter).Decode(&finalShoppingCartItem)
		if fetchErr != nil {
			log.Printf("Failed to fetch updated shopping cart item: %v", fetchErr)
			http.Error(w, "Failed to fetch updated item", http.StatusInternalServerError)
			return
		}

	} else if errors.Is(err, mongo.ErrNoDocuments) {
		// Item not found - Insert new item
		itemWasUpdated = false
		newItem := models.CreateShoppingCartItem(
			userID,
			user.GroupID,
			request.ItemName,
			request.Quantity,
			request.Category,
		)
		insertResult, insertErr := config.DB.Collection("shopping_cart").InsertOne(context.Background(), newItem)
		if insertErr != nil {
			log.Printf("Failed to insert new shopping cart item: %v", insertErr)
			http.Error(w, "Failed to add item to shopping cart", http.StatusInternalServerError)
			return
		}
		newItem.ID = insertResult.InsertedID.(primitive.ObjectID)
		finalShoppingCartItem = *newItem // Use the newly inserted item data (Dereference the pointer)

	} else {
		// Other database error during FindOne
		log.Printf("Error checking for existing shopping cart item: %v", err)
		http.Error(w, "Database error checking for item", http.StatusInternalServerError)
		return
	}

	// Log the activity
	go func() {
		activityAction := models.CartActivityTypeAdd
		activityDetails := "Added item to shopping cart"
		if itemWasUpdated {
			activityAction = models.CartActivityTypeUpdate // Using Update type for increment as well
			activityDetails = fmt.Sprintf("Increased quantity of %s by %.2f (New total: %.2f)", finalShoppingCartItem.ItemName, request.Quantity, finalShoppingCartItem.Quantity)
		}

		activity := models.CreateShoppingCartActivity(
			user.GroupID,
			finalShoppingCartItem.ID, // Use the ID from the final item state
			finalShoppingCartItem.ItemName,
			userID,
			user.Name,
			activityAction,                 // Use the determined action
			finalShoppingCartItem.Quantity, // Log the *new* total quantity
			activityDetails,                // Use the determined details
		)

		_, insertErr := config.DB.Collection("shopping_cart_activity").InsertOne(
			context.Background(),
			activity,
		)

		if insertErr != nil {
			log.Printf("Failed to create shopping cart activity record: %v", insertErr)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK for both add and increment
	json.NewEncoder(w).Encode(ShoppingCartResponse{
		Status:  "success",
		Message: "Item processed successfully", // Updated generic message
		Data:    finalShoppingCartItem,
	})
}

// UpdateShoppingCartItemHandler handles updating an item in the shopping cart
func UpdateShoppingCartItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var request UpdateShoppingCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ItemID == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	// Convert item ID to ObjectID
	itemID, err := primitive.ObjectIDFromHex(request.ItemID)
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

	// Verify the item belongs to the user
	var shoppingCartItem models.ShoppingCartItem
	err = config.DB.Collection("shopping_cart").FindOne(
		context.Background(),
		bson.M{
			"_id":     itemID,
			"user_id": userID,
		},
	).Decode(&shoppingCartItem)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Shopping cart item not found or does not belong to user", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch shopping cart item", http.StatusInternalServerError)
		}
		return
	}

	// Prepare update fields
	updateFields := bson.M{}

	// Only update fields that were provided
	if request.ItemName != "" {
		updateFields["item_name"] = request.ItemName
	}

	if request.Quantity > 0 {
		updateFields["quantity"] = request.Quantity
	}

	if request.Category != "" {
		updateFields["category"] = request.Category
	}

	// If no fields to update, return early
	if len(updateFields) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	// Record the old values for activity logging
	oldItemName := shoppingCartItem.ItemName
	oldQuantity := shoppingCartItem.Quantity

	// Update the item
	_, err = config.DB.Collection("shopping_cart").UpdateOne(
		context.Background(),
		bson.M{"_id": itemID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("Failed to update shopping cart item: %v", err)
		http.Error(w, "Failed to update shopping cart item", http.StatusInternalServerError)
		return
	}

	// Get updated item
	err = config.DB.Collection("shopping_cart").FindOne(
		context.Background(),
		bson.M{"_id": itemID},
	).Decode(&shoppingCartItem)

	if err != nil {
		log.Printf("Failed to fetch updated shopping cart item: %v", err)
		http.Error(w, "Failed to fetch updated item", http.StatusInternalServerError)
		return
	}

	// Log the activity
	go func() {
		// Create details message
		details := "Updated item in shopping cart: "
		changes := []string{}

		if request.ItemName != "" && request.ItemName != oldItemName {
			changes = append(changes, "name from '"+oldItemName+"' to '"+request.ItemName+"'")
		}

		if request.Quantity > 0 && request.Quantity != oldQuantity {
			changes = append(changes, "quantity from "+fmt.Sprintf("%.2f", oldQuantity)+" to "+fmt.Sprintf("%.2f", request.Quantity))
		}

		details += strings.Join(changes, ", ")

		// Create activity log
		activity := models.CreateShoppingCartActivity(
			user.GroupID,
			itemID,
			shoppingCartItem.ItemName,
			userID,
			user.Name,
			models.CartActivityTypeUpdate,
			shoppingCartItem.Quantity,
			details,
		)

		_, err := config.DB.Collection("shopping_cart_activity").InsertOne(
			context.Background(),
			activity,
		)

		if err != nil {
			log.Printf("Failed to create shopping cart activity record: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ShoppingCartResponse{
		Status:  "success",
		Message: "Item updated in shopping cart",
		Data:    shoppingCartItem,
	})
}

// DeleteShoppingCartItemHandler handles deleting an item from the shopping cart
func DeleteShoppingCartItemHandler(w http.ResponseWriter, r *http.Request) {
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
	itemIDStr := strings.TrimPrefix(r.URL.Path, "/api/shopping-cart/delete/")
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

	// Get the item before deletion for logging purposes
	var shoppingCartItem models.ShoppingCartItem
	err = config.DB.Collection("shopping_cart").FindOne(
		context.Background(),
		bson.M{
			"_id":     itemID,
			"user_id": userID,
		},
	).Decode(&shoppingCartItem)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Shopping cart item not found or does not belong to user", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch shopping cart item", http.StatusInternalServerError)
		}
		return
	}

	// Delete the item if it belongs to the user
	result, err := config.DB.Collection("shopping_cart").DeleteOne(
		context.Background(),
		bson.M{
			"_id":     itemID,
			"user_id": userID,
		},
	)

	if err != nil {
		log.Printf("Failed to delete shopping cart item: %v", err)
		http.Error(w, "Failed to delete shopping cart item", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Shopping cart item not found or does not belong to user", http.StatusNotFound)
		return
	}

	// Log the activity
	go func() {
		// Create activity log
		activity := models.CreateShoppingCartActivity(
			user.GroupID,
			itemID,
			shoppingCartItem.ItemName,
			userID,
			user.Name,
			models.CartActivityTypeDelete,
			shoppingCartItem.Quantity,
			"Removed item from shopping cart",
		)

		_, err := config.DB.Collection("shopping_cart_activity").InsertOne(
			context.Background(),
			activity,
		)

		if err != nil {
			log.Printf("Failed to create shopping cart activity record: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ShoppingCartResponse{
		Status:  "success",
		Message: "Shopping cart item deleted successfully",
	})
}

// ListShoppingCartItemsHandler handles retrieving all items in the shopping cart for a group
func ListShoppingCartItemsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Check for filter by user
	filterByUser := r.URL.Query().Get("user_id")

	// Build the query filter
	filter := bson.M{"group_id": user.GroupID}

	// If filtering by user, add user_id to filter
	if filterByUser != "" {
		filterUserID, err := primitive.ObjectIDFromHex(filterByUser)
		if err != nil {
			http.Error(w, "Invalid filter user ID format", http.StatusBadRequest)
			return
		}

		// Verify the filter user belongs to the same group
		var filterUser models.User
		err = config.DB.Collection("users").FindOne(
			context.Background(),
			bson.M{"_id": filterUserID},
		).Decode(&filterUser)

		if err != nil || filterUser.GroupID != user.GroupID {
			http.Error(w, "User not found or not in your group", http.StatusForbidden)
			return
		}

		filter["user_id"] = filterUserID
	}

	// Get all items in the shopping cart for the group
	opts := options.Find().SetSort(bson.D{{Key: "added_at", Value: -1}})
	cursor, err := config.DB.Collection("shopping_cart").Find(
		context.Background(),
		filter,
		opts,
	)

	if err != nil {
		log.Printf("Failed to fetch shopping cart items: %v", err)
		http.Error(w, "Failed to fetch shopping cart items", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var shoppingCartItems []models.ShoppingCartItem
	if err = cursor.All(context.Background(), &shoppingCartItems); err != nil {
		log.Printf("Failed to decode shopping cart items: %v", err)
		http.Error(w, "Failed to decode shopping cart items", http.StatusInternalServerError)
		return
	}

	// Return items with additional user info
	type ShoppingCartItemWithUser struct {
		models.ShoppingCartItem
		UserName string `json:"user_name"`
	}

	// Create a map of user IDs to user names
	userCache := make(map[string]string)

	// Prepare response with user names
	itemsWithUsers := make([]ShoppingCartItemWithUser, 0, len(shoppingCartItems))
	for _, item := range shoppingCartItems {
		userIDStr := item.UserID.Hex()
		userName, ok := userCache[userIDStr]

		if !ok {
			// Fetch user name if not in cache
			var itemUser models.User
			err := config.DB.Collection("users").FindOne(
				context.Background(),
				bson.M{"_id": item.UserID},
			).Decode(&itemUser)

			if err == nil {
				userName = itemUser.Name
				userCache[userIDStr] = userName
			} else {
				userName = "Unknown User"
			}
		}

		itemWithUser := ShoppingCartItemWithUser{
			ShoppingCartItem: item,
			UserName:         userName,
		}

		itemsWithUsers = append(itemsWithUsers, itemWithUser)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShoppingCartResponse{
		Status:  "success",
		Message: "Shopping cart items retrieved successfully",
		Data:    itemsWithUsers,
	})
}

// GetShoppingCartActivityHandler retrieves recent activity for a group's shopping cart
func GetShoppingCartActivityHandler(w http.ResponseWriter, r *http.Request) {
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
	limit := 20 // Default limit

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

	// Set up options for sorting and limiting results
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}). // Sort by newest first
		SetLimit(int64(limit))

	// Find all activity for this group
	cursor, err := config.DB.Collection("shopping_cart_activity").Find(
		context.Background(),
		bson.M{"group_id": group.ID},
		opts,
	)

	if err != nil {
		http.Error(w, "Failed to fetch shopping cart activity", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var activities []models.ShoppingCartActivity
	if err = cursor.All(context.Background(), &activities); err != nil {
		http.Error(w, "Failed to decode shopping cart activity", http.StatusInternalServerError)
		return
	}

	// Update read status for the current user
	go func() {
		for _, activity := range activities {
			if !activity.HasBeenReadBy(userID) {
				_, err := config.DB.Collection("shopping_cart_activity").UpdateOne(
					context.Background(),
					bson.M{"_id": activity.ID},
					bson.M{"$addToSet": bson.M{"read_by": userID}},
				)

				if err != nil {
					log.Printf("Failed to update activity read status: %v", err)
				}
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShoppingCartResponse{
		Status:  "success",
		Message: "Shopping cart activity retrieved successfully",
		Data:    activities,
	})
}

// MarkActivityReadHandler marks a shopping cart activity as read
func MarkActivityReadHandler(w http.ResponseWriter, r *http.Request) {
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
		ActivityID string `json:"activity_id" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ActivityID == "" {
		http.Error(w, "Activity ID is required", http.StatusBadRequest)
		return
	}

	// Convert activity ID to ObjectID
	activityID, err := primitive.ObjectIDFromHex(request.ActivityID)
	if err != nil {
		http.Error(w, "Invalid activity ID format", http.StatusBadRequest)
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

	// Find activity
	var activity models.ShoppingCartActivity
	err = config.DB.Collection("shopping_cart_activity").FindOne(
		context.Background(),
		bson.M{"_id": activityID},
	).Decode(&activity)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Activity not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch activity", http.StatusInternalServerError)
		}
		return
	}

	// Verify user belongs to the activity's group
	if user.GroupID != activity.GroupID {
		http.Error(w, "User is not a member of this activity's group", http.StatusForbidden)
		return
	}

	// Mark activity as read - update both the read_by array and the is_read flag
	_, err = config.DB.Collection("shopping_cart_activity").UpdateOne(
		context.Background(),
		bson.M{"_id": activityID},
		bson.M{
			"$addToSet": bson.M{"read_by": userID},
			"$set":      bson.M{"is_read": true},
		},
	)

	if err != nil {
		http.Error(w, "Failed to mark activity as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Activity marked as read",
	})
}
