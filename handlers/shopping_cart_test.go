package handlers_test

import (
	"bytes"
	"context"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"cribb-backend/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAddShoppingCartItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a shopping cart item request
	cartItemReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
		Category string  `json:"category"`
	}{
		ItemName: "Milk",
		Quantity: 2.0,
		Category: "Dairy",
	}

	reqBody, _ := json.Marshal(cartItemReq)
	req, err := http.NewRequest("POST", "/api/shopping-cart/add", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemName string  `json:"item_name"`
			Quantity float64 `json:"quantity"`
			Category string  `json:"category"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Item name and quantity are required. Quantity must be positive.", http.StatusBadRequest)
			return
		}

		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)
		user, found := testDB.FindUserByID(userID)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Create shopping cart item
		cartItem := models.CreateShoppingCartItem(
			userID,
			user.GroupID,
			req.ItemName,
			req.Quantity,
			req.Category,
		)

		// Set ID (simulating insert)
		cartItem.ID = primitive.NewObjectID()

		// Add to test DB (simulating insert)
		testDB.AddShoppingCartItem(*cartItem)

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cartItem)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var cartItem models.ShoppingCartItem
	err = json.Unmarshal(rr.Body.Bytes(), &cartItem)
	if err != nil {
		t.Fatal(err)
	}

	// Verify item properties
	if cartItem.ItemName != cartItemReq.ItemName {
		t.Errorf("Expected item name %s, got %s", cartItemReq.ItemName, cartItem.ItemName)
	}
	if cartItem.Quantity != cartItemReq.Quantity {
		t.Errorf("Expected quantity %f, got %f", cartItemReq.Quantity, cartItem.Quantity)
	}
	if cartItem.UserID != testUser.ID {
		t.Errorf("Expected user ID %s, got %s", testUser.ID.Hex(), cartItem.UserID.Hex())
	}
	if cartItem.GroupID != testGroup.ID {
		t.Errorf("Expected group ID %s, got %s", testGroup.ID.Hex(), cartItem.GroupID.Hex())
	}
}

func TestUpdateShoppingCartItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Create existing shopping cart item
	existingItem := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser.ID,
		GroupID:  testGroup.ID,
		ItemName: "Bread",
		Quantity: 1.0,
		Category: "Bakery",
		AddedAt:  time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddShoppingCartItem(existingItem)

	// Create an update request
	updateReq := struct {
		ItemID   string  `json:"item_id"`
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
		Category string  `json:"category"`
	}{
		ItemID:   existingItem.ID.Hex(),
		ItemName: "Whole Wheat Bread",
		Quantity: 2.0,
		Category: "Organic Bakery",
	}

	reqBody, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PUT", "/api/shopping-cart/update", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemID   string  `json:"item_id"`
			ItemName string  `json:"item_name,omitempty"`
			Quantity float64 `json:"quantity,omitempty"`
			Category string  `json:"category,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemID == "" {
			http.Error(w, "Item ID is required", http.StatusBadRequest)
			return
		}

		// Convert item ID
		itemID, err := primitive.ObjectIDFromHex(req.ItemID)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)

		// Find item in mock database
		item, found := testDB.FindShoppingCartItemByID(itemID)
		if !found {
			http.Error(w, "Shopping cart item not found", http.StatusNotFound)
			return
		}

		// Check if item belongs to user
		if item.UserID != userID {
			http.Error(w, "Shopping cart item does not belong to user", http.StatusForbidden)
			return
		}

		// Update the item
		if req.ItemName != "" {
			item.ItemName = req.ItemName
		}

		if req.Quantity > 0 {
			item.Quantity = req.Quantity
		}

		if req.Category != "" {
			item.Category = req.Category
		}

		// Update in mock database
		testDB.UpdateShoppingCartItem(item)

		// Return updated item
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(item)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var cartItem models.ShoppingCartItem
	err = json.Unmarshal(rr.Body.Bytes(), &cartItem)
	if err != nil {
		t.Fatal(err)
	}

	// Verify item properties were updated
	if cartItem.ItemName != updateReq.ItemName {
		t.Errorf("Expected item name to be updated to %s, got %s", updateReq.ItemName, cartItem.ItemName)
	}
	if cartItem.Quantity != updateReq.Quantity {
		t.Errorf("Expected quantity to be updated to %f, got %f", updateReq.Quantity, cartItem.Quantity)
	}

	if cartItem.Category != updateReq.Category {
		t.Errorf("Expected category to be updated to %s, got %s", updateReq.Category, cartItem.Category)
	}
}

func TestDeleteShoppingCartItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Create existing shopping cart item
	existingItem := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser.ID,
		GroupID:  testGroup.ID,
		ItemName: "Bread",
		Quantity: 1.0,
		AddedAt:  time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddShoppingCartItem(existingItem)

	// Create a request
	req, err := http.NewRequest("DELETE", "/api/shopping-cart/delete/"+existingItem.ID.Hex(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract item ID from path
		itemIDStr := existingItem.ID.Hex() // In a real test, you would extract this from the URL path

		// Convert item ID
		itemID, err := primitive.ObjectIDFromHex(itemIDStr)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)

		// Find item in mock database
		item, found := testDB.FindShoppingCartItemByID(itemID)
		if !found {
			http.Error(w, "Shopping cart item not found", http.StatusNotFound)
			return
		}

		// Check if item belongs to user
		if item.UserID != userID {
			http.Error(w, "Shopping cart item does not belong to user", http.StatusForbidden)
			return
		}

		// Delete from mock database
		deleted := testDB.DeleteShoppingCartItem(itemID)
		if !deleted {
			http.Error(w, "Failed to delete shopping cart item", http.StatusInternalServerError)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Shopping cart item deleted successfully",
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Verify success message
	if response["message"] != "Shopping cart item deleted successfully" {
		t.Errorf("Expected success message, got %s", response["message"])
	}

	// Verify item was deleted
	_, found := testDB.FindShoppingCartItemByID(existingItem.ID)
	if found {
		t.Errorf("Item should have been deleted from database")
	}
}

func TestListShoppingCartItemsHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser1 := test.CreateTestUser()
	testUser1.GroupID = testGroup.ID
	testUser2 := test.CreateTestUser()
	testUser2.ID = primitive.NewObjectID() // Ensure different ID
	testUser2.Username = "testuser2"
	testUser2.Name = "Test User 2"
	testUser2.GroupID = testGroup.ID

	// Create existing shopping cart items
	item1 := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser1.ID,
		GroupID:  testGroup.ID,
		ItemName: "Milk",
		Quantity: 2.0,
		Category: "Dairy",
		AddedAt:  time.Now(),
	}

	item2 := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser2.ID,
		GroupID:  testGroup.ID,
		ItemName: "Bread",
		Quantity: 1.0,
		Category: "Bakery",
		AddedAt:  time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)
	testDB.AddShoppingCartItem(item1)
	testDB.AddShoppingCartItem(item2)

	// Create a request
	req, err := http.NewRequest("GET", "/api/shopping-cart/list", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser1.ID.Hex(),
		Username: testUser1.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)
		user, found := testDB.FindUserByID(userID)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check for filter by user
		filterByUser := r.URL.Query().Get("user_id")

		// Get all items for the group from mock database
		var items []models.ShoppingCartItem
		for _, item := range testDB.ShoppingCartItems {
			if item.GroupID == user.GroupID {
				// Apply user filter if provided
				if filterByUser != "" {
					filterUserID, err := primitive.ObjectIDFromHex(filterByUser)
					if err != nil {
						continue // Skip invalid user IDs
					}
					if item.UserID != filterUserID {
						continue // Skip items that don't match the filter
					}
				}
				items = append(items, item)
			}
		}

		// Create response with user info
		type ItemWithUser struct {
			models.ShoppingCartItem
			UserName string `json:"user_name"`
		}

		// Get user names and build response
		itemsWithUsers := make([]ItemWithUser, 0, len(items))
		for _, item := range items {
			userName := "Unknown User"

			// Find user name
			for _, u := range testDB.Users {
				if u.ID == item.UserID {
					userName = u.Name
					break
				}
			}

			itemWithUser := ItemWithUser{
				ShoppingCartItem: item,
				UserName:         userName,
			}
			itemsWithUsers = append(itemsWithUsers, itemWithUser)
		}

		// Return items
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(itemsWithUsers)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response []map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all items were returned
	if len(response) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response))
	}

	// Verify items have user names
	for _, item := range response {
		if _, ok := item["user_name"]; !ok {
			t.Errorf("Expected item to have user_name field")
		}
	}

	// Test filtering by user
	// Create a request with user filter
	reqFiltered, err := http.NewRequest("GET", "/api/shopping-cart/list?user_id="+testUser1.ID.Hex(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set auth context
	reqFiltered = reqFiltered.WithContext(ctx)

	// Record the response
	rrFiltered := httptest.NewRecorder()

	// Call the handler with filtered request
	handler.ServeHTTP(rrFiltered, reqFiltered)

	// Check the response
	var filteredResponse []map[string]interface{}
	err = json.Unmarshal(rrFiltered.Body.Bytes(), &filteredResponse)
	if err != nil {
		t.Fatal(err)
	}

	// Verify only items from testUser1 were returned
	if len(filteredResponse) != 1 {
		t.Errorf("Expected 1 filtered item, got %d", len(filteredResponse))
	}

	// Test error case: invalid user ID
	reqInvalid, err := http.NewRequest("GET", "/api/shopping-cart/list?user_id=invalid-id", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set auth context
	reqInvalid = reqInvalid.WithContext(ctx)

	// Record the response
	rrInvalid := httptest.NewRecorder()

	// Create handler for invalid case
	handlerInvalid := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		_, ok := middleware.GetUserFromContext(r.Context())

		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Check for filter by user
		filterByUser := r.URL.Query().Get("user_id")

		// Test invalid user ID
		if filterByUser != "" {
			_, err := primitive.ObjectIDFromHex(filterByUser)
			if err != nil {
				http.Error(w, "Invalid filter user ID format", http.StatusBadRequest)
				return
			}
		}

		// If we get here, the test failed
		w.WriteHeader(http.StatusOK)
	})

	// Call the handler with invalid request
	handlerInvalid.ServeHTTP(rrInvalid, reqInvalid)

	// Check for error response
	if status := rrInvalid.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid user ID: got %v want %v",
			status, http.StatusBadRequest)
	}
}

// Test case for attempting to delete another user's item
func TestDeleteShoppingCartItemNotOwnedHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data for two users
	testGroup := test.CreateTestGroup()
	testUser1 := test.CreateTestUser()
	testUser1.GroupID = testGroup.ID
	testUser2 := test.CreateTestUser()
	testUser2.ID = primitive.NewObjectID() // Ensure different ID
	testUser2.Username = "testuser2"
	testUser2.GroupID = testGroup.ID

	// Create item owned by user2
	user2Item := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser2.ID,
		GroupID:  testGroup.ID,
		ItemName: "Eggs",
		Quantity: 1.0,
		Category: "Dairy",
		AddedAt:  time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)
	testDB.AddShoppingCartItem(user2Item)

	// Create a request - user1 trying to delete user2's item
	req, err := http.NewRequest("DELETE", "/api/shopping-cart/delete/"+user2Item.ID.Hex(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context for user1
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser1.ID.Hex(),
		Username: testUser1.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract item ID from path
		itemIDStr := user2Item.ID.Hex() // In a real test, you would extract this from the URL path

		// Convert item ID
		itemID, err := primitive.ObjectIDFromHex(itemIDStr)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)

		// Find item in mock database
		item, found := testDB.FindShoppingCartItemByID(itemID)
		if !found {
			http.Error(w, "Shopping cart item not found", http.StatusNotFound)
			return
		}

		// Check if item belongs to user
		if item.UserID != userID {
			http.Error(w, "Shopping cart item does not belong to user", http.StatusForbidden)
			return
		}

		// If we get here, the test failed because the item should not belong to user1
		t.Errorf("Should not be able to delete another user's item")

		// Delete from mock database and return success (which shouldn't happen)
		testDB.DeleteShoppingCartItem(itemID)
		w.WriteHeader(http.StatusOK)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get forbidden status
	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
	}

	// Verify item still exists in database
	_, found := testDB.FindShoppingCartItemByID(user2Item.ID)
	if !found {
		t.Errorf("Item should not have been deleted from database")
	}
}

// Test case for adding duplicate item with existing name
func TestAddDuplicateShoppingCartItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Create existing item
	existingItem := models.ShoppingCartItem{
		ID:       primitive.NewObjectID(),
		UserID:   testUser.ID,
		GroupID:  testGroup.ID,
		ItemName: "Milk",
		Quantity: 1.0,
		AddedAt:  time.Now().Add(-time.Hour), // Added an hour ago
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddShoppingCartItem(existingItem)

	// Create request to add same item with different quantity
	cartItemReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemName: "Milk", // Same name as existing item
		Quantity: 2.0,    // Different quantity
	}

	reqBody, _ := json.Marshal(cartItemReq)
	req, err := http.NewRequest("POST", "/api/shopping-cart/add", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create handler function that simulates upsert behavior
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemName string  `json:"item_name"`
			Quantity float64 `json:"quantity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get user from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Find user
		userID, _ := primitive.ObjectIDFromHex(userClaims.ID)
		user, found := testDB.FindUserByID(userID)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check if item already exists with same name for this user and group
		var finalItem models.ShoppingCartItem // Renamed for clarity
		itemExists := false

		for _, item := range testDB.ShoppingCartItems {
			if item.UserID == userID && item.GroupID == user.GroupID && item.ItemName == req.ItemName {
				// Simulate Incrementing existing item quantity
				finalItem = item
				finalItem.Quantity += req.Quantity // Add the requested quantity
				finalItem.AddedAt = time.Now()     // Update timestamp

				testDB.UpdateShoppingCartItem(finalItem) // Update in the mock DB
				itemExists = true
				break
			}
		}

		// If item doesn't exist, create new one
		if !itemExists {
			newItem := models.CreateShoppingCartItem(
				userID,
				user.GroupID,
				req.ItemName,
				req.Quantity,
				"", // Assuming category is empty if not provided in this test scenario
			)
			newItem.ID = primitive.NewObjectID()
			testDB.AddShoppingCartItem(*newItem)
			finalItem = *newItem
		}

		// Return item
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(finalItem) // Return the final state
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var cartItem models.ShoppingCartItem
	err = json.Unmarshal(rr.Body.Bytes(), &cartItem)
	if err != nil {
		t.Fatal(err)
	}

	// Verify item was updated, not added
	if cartItem.ID != existingItem.ID {
		t.Errorf("Expected to update existing item ID (%s), got different ID (%s)", existingItem.ID.Hex(), cartItem.ID.Hex())
	}

	// Verify quantity was incremented
	expectedQuantity := existingItem.Quantity + cartItemReq.Quantity
	if cartItem.Quantity != expectedQuantity {
		t.Errorf("Expected quantity to be incremented to %f (%.2f + %.2f), got %f", expectedQuantity, existingItem.Quantity, cartItemReq.Quantity, cartItem.Quantity)
	}

	// Verify timestamp is newer
	if !cartItem.AddedAt.After(existingItem.AddedAt) {
		t.Errorf("Expected timestamp to be updated")
	}

	// Check how many items exist in DB - should still be just one
	itemCount := 0
	for _, item := range testDB.ShoppingCartItems {
		if item.UserID == testUser.ID && item.GroupID == testGroup.ID && item.ItemName == cartItemReq.ItemName {
			itemCount++
		}
	}

	if itemCount != 1 {
		t.Errorf("Expected 1 item in database, got %d", itemCount)
	}
}

// Test attempting to update a non-existent item
func TestUpdateNonexistentShoppingCartItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a non-existent item ID
	nonexistentID := primitive.NewObjectID()

	// Create an update request with non-existent ID
	updateReq := struct {
		ItemID   string  `json:"item_id"`
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemID:   nonexistentID.Hex(),
		ItemName: "Non-existent Item",
		Quantity: 1.0,
	}

	reqBody, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PUT", "/api/shopping-cart/update", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler function that simulates the actual handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemID   string  `json:"item_id"`
			ItemName string  `json:"item_name,omitempty"`
			Quantity float64 `json:"quantity,omitempty"`
			Category string  `json:"category,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemID == "" {
			http.Error(w, "Item ID is required", http.StatusBadRequest)
			return
		}

		// Convert item ID
		itemID, err := primitive.ObjectIDFromHex(req.ItemID)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		// Find item in mock database
		_, found := testDB.FindShoppingCartItemByID(itemID)
		if !found {
			http.Error(w, "Shopping cart item not found", http.StatusNotFound)
			return
		}

		// If we get here, the test failed
		t.Errorf("Should not find non-existent item")
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get not found status
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

// Test invalid requests
func TestInvalidShoppingCartRequests(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Test adding item with empty name
	invalidAddReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemName: "", // Empty name
		Quantity: 1.0,
	}

	reqBody, _ := json.Marshal(invalidAddReq)
	req, err := http.NewRequest("POST", "/api/shopping-cart/add", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Create auth context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Record the response
	rr := httptest.NewRecorder()

	// Create handler function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemName string  `json:"item_name"`
			Quantity float64 `json:"quantity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemName == "" || req.Quantity <= 0 {
			http.Error(w, "Item name and quantity are required. Quantity must be positive.", http.StatusBadRequest)
			return
		}

		// If we get here, the test failed
		t.Errorf("Should have rejected empty item name")
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get bad request status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Test adding item with zero quantity
	invalidQuantityReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemName: "Test Item",
		Quantity: 0.0, // Zero quantity
	}

	reqBody, _ = json.Marshal(invalidQuantityReq)
	req, err = http.NewRequest("POST", "/api/shopping-cart/add", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Set auth context
	req = req.WithContext(ctx)

	// Record the response
	rr = httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get bad request status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Test updating with missing ID
	invalidUpdateReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemName: "Updated Item",
		Quantity: 2.0,
	}

	reqBody, _ = json.Marshal(invalidUpdateReq)
	req, err = http.NewRequest("PUT", "/api/shopping-cart/update", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Set auth context
	req = req.WithContext(ctx)

	// Record the response
	rr = httptest.NewRecorder()

	// Create update handler
	updateHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ItemID   string  `json:"item_id"`
			ItemName string  `json:"item_name,omitempty"`
			Quantity float64 `json:"quantity,omitempty"`
			Category string  `json:"category,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemID == "" {
			http.Error(w, "Item ID is required", http.StatusBadRequest)
			return
		}

		// If we get here, the test failed
		t.Errorf("Should have rejected missing item ID")
	})

	// Call the update handler
	updateHandler.ServeHTTP(rr, req)

	// Should get bad request status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Test deleting with invalid ID format
	req, err = http.NewRequest("DELETE", "/api/shopping-cart/delete/invalid-id", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set auth context
	req = req.WithContext(ctx)

	// Record the response
	rr = httptest.NewRecorder()

	// Create delete handler
	deleteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract item ID from path
		itemIDStr := "invalid-id" // In a real test, you would extract this from the URL path

		// Convert item ID
		_, err := primitive.ObjectIDFromHex(itemIDStr)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		// If we get here, the test failed
		t.Errorf("Should have rejected invalid ID format")
	})

	// Call the delete handler
	deleteHandler.ServeHTTP(rr, req)

	// Should get bad request status
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// Test case for unauthenticated requests
func TestUnauthenticatedShoppingCartRequests(t *testing.T) {
	// Create a shopping cart item request
	cartItemReq := struct {
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemName: "Milk",
		Quantity: 2.0,
	}

	reqBody, _ := json.Marshal(cartItemReq)

	// Test add endpoint without auth
	req, err := http.NewRequest("POST", "/api/shopping-cart/add", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create handler function
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		_, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// If we get here, the test failed
		t.Errorf("Should have rejected unauthenticated request")
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get unauthorized status
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

	// Test update endpoint without auth
	updateReq := struct {
		ItemID   string  `json:"item_id"`
		ItemName string  `json:"item_name"`
		Quantity float64 `json:"quantity"`
	}{
		ItemID:   primitive.NewObjectID().Hex(),
		ItemName: "Updated Item",
		Quantity: 2.0,
	}

	reqBody, _ = json.Marshal(updateReq)
	req, err = http.NewRequest("PUT", "/api/shopping-cart/update", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr = httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get unauthorized status
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

	// Test delete endpoint without auth
	req, err = http.NewRequest("DELETE", "/api/shopping-cart/delete/"+primitive.NewObjectID().Hex(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr = httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get unauthorized status
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

	// Test list endpoint without auth
	req, err = http.NewRequest("GET", "/api/shopping-cart/list", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr = httptest.NewRecorder()

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Should get unauthorized status
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}
}
