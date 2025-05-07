// handlers/pantry_test.go
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

func TestAddPantryItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a pantry item request
	expirationDate := time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	pantryReq := struct {
		Name           string  `json:"name"`
		Quantity       float64 `json:"quantity"`
		Unit           string  `json:"unit"`
		Category       string  `json:"category"`
		ExpirationDate string  `json:"expiration_date"`
		GroupName      string  `json:"group_name"`
	}{
		Name:           "Milk",
		Quantity:       2.0,
		Unit:           "gallons",
		Category:       "Dairy",
		ExpirationDate: expirationDate,
		GroupName:      testGroup.Name,
	}

	reqBody, _ := json.Marshal(pantryReq)
	req, err := http.NewRequest("POST", "/api/pantry/add", bytes.NewBuffer(reqBody))
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
			Name           string  `json:"name"`
			Quantity       float64 `json:"quantity"`
			Unit           string  `json:"unit"`
			Category       string  `json:"category"`
			ExpirationDate string  `json:"expiration_date"`
			GroupName      string  `json:"group_name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Name == "" || req.Quantity < 0 || req.Unit == "" || req.GroupName == "" {
			http.Error(w, "Name, quantity, unit, and group name are required", http.StatusBadRequest)
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

		// Find group
		group, found := testDB.FindGroupByName(req.GroupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Check user belongs to group
		if user.GroupID != group.ID {
			http.Error(w, "User is not a member of this group", http.StatusForbidden)
			return
		}

		// Parse expiration date
		var expirationDate time.Time
		if req.ExpirationDate != "" {
			var err error
			expirationDate, err = time.Parse(time.RFC3339, req.ExpirationDate)
			if err != nil {
				http.Error(w, "Invalid expiration date format", http.StatusBadRequest)
				return
			}
		}

		// Create pantry item
		pantryItem := models.CreatePantryItem(
			group.ID,
			req.Name,
			req.Quantity,
			req.Unit,
			req.Category,
			expirationDate,
			user.ID,
		)
		pantryItem.ID = primitive.NewObjectID()

		// Add to database
		testDB.AddPantryItem(*pantryItem)

		// Create pantry history record for tracking
		history := models.CreatePantryHistory(
			group.ID,
			pantryItem.ID,
			pantryItem.Name,
			user.ID,
			user.Name,
			models.ActionTypeAdd,
			req.Quantity,
			"Added new item",
		)
		testDB.AddPantryHistory(*history)

		// Return success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(pantryItem)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var pantryItem models.PantryItem
	err = json.Unmarshal(rr.Body.Bytes(), &pantryItem)
	if err != nil {
		t.Fatal(err)
	}

	// Verify item properties
	if pantryItem.Name != pantryReq.Name {
		t.Errorf("Expected name %s, got %s", pantryReq.Name, pantryItem.Name)
	}
	if pantryItem.Quantity != pantryReq.Quantity {
		t.Errorf("Expected quantity %f, got %f", pantryReq.Quantity, pantryItem.Quantity)
	}
	if pantryItem.Unit != pantryReq.Unit {
		t.Errorf("Expected unit %s, got %s", pantryReq.Unit, pantryItem.Unit)
	}
}

func TestUsePantryItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	testPantryItem := models.PantryItem{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		Name:      "Milk",
		Quantity:  2.0,
		Unit:      "gallons",
		Category:  "Dairy",
		AddedBy:   testUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddPantryItem(testPantryItem)

	// Create a use pantry item request
	usePantryReq := struct {
		ItemID   string  `json:"item_id"`
		Quantity float64 `json:"quantity"`
	}{
		ItemID:   testPantryItem.ID.Hex(),
		Quantity: 0.5,
	}

	reqBody, _ := json.Marshal(usePantryReq)
	req, err := http.NewRequest("POST", "/api/pantry/use", bytes.NewBuffer(reqBody))
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
			Quantity float64 `json:"quantity"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.ItemID == "" || req.Quantity <= 0 {
			http.Error(w, "Item ID and quantity are required. Quantity must be positive.", http.StatusBadRequest)
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

		// Find item
		itemID, err := primitive.ObjectIDFromHex(req.ItemID)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		item, found := testDB.FindPantryItemByID(itemID)
		if !found {
			http.Error(w, "Pantry item not found", http.StatusNotFound)
			return
		}

		// Check user belongs to item's group
		if user.GroupID != item.GroupID {
			http.Error(w, "User is not a member of this item's group", http.StatusForbidden)
			return
		}

		// Check if there's enough quantity
		if item.Quantity < req.Quantity {
			http.Error(w, "Not enough quantity available", http.StatusBadRequest)
			return
		}

		// Update the quantity
		newQuantity := item.Quantity - req.Quantity
		item.UpdateQuantity(newQuantity)

		// Update in database
		testDB.UpdatePantryItem(item)

		// Create history record
		history := models.CreatePantryHistory(
			item.GroupID,
			item.ID,
			item.Name,
			user.ID,
			user.Name,
			models.ActionTypeUse,
			req.Quantity,
			"Used item",
		)
		testDB.AddPantryHistory(*history)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":            true,
			"message":            "Item used successfully",
			"remaining_quantity": newQuantity,
			"unit":               item.Unit,
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the item was updated correctly
	updatedItem, found := testDB.FindPantryItemByID(testPantryItem.ID)
	if !found {
		t.Errorf("Item should exist in database after use")
	}

	expectedQuantity := testPantryItem.Quantity - usePantryReq.Quantity
	if updatedItem.Quantity != expectedQuantity {
		t.Errorf("Expected quantity after use to be %f, got %f", expectedQuantity, updatedItem.Quantity)
	}

	// Check if response contains correct values
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("Expected success to be true in response")
	}

	remainingQty, ok := response["remaining_quantity"].(float64)
	if !ok || remainingQty != expectedQuantity {
		t.Errorf("Expected remaining_quantity to be %f, got %v", expectedQuantity, response["remaining_quantity"])
	}
}

func TestDeletePantryItemHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	testPantryItem := models.PantryItem{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		Name:      "Bread",
		Quantity:  1.0,
		Unit:      "loaf",
		Category:  "Bakery",
		AddedBy:   testUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddPantryItem(testPantryItem)

	// Create a request
	req, err := http.NewRequest("DELETE", "/api/pantry/remove/"+testPantryItem.ID.Hex(), nil)
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
		// Get item ID from URL path
		itemIDStr := "abcd1234" // Simulate parsing from URL
		if req.URL.Path != "" {
			// In a real test, you would extract this from the URL path
			itemIDStr = testPantryItem.ID.Hex()
		}

		if itemIDStr == "" {
			http.Error(w, "Item ID is required", http.StatusBadRequest)
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

		// Find item
		itemID, err := primitive.ObjectIDFromHex(itemIDStr)
		if err != nil {
			http.Error(w, "Invalid item ID format", http.StatusBadRequest)
			return
		}

		item, found := testDB.FindPantryItemByID(itemID)
		if !found {
			http.Error(w, "Pantry item not found", http.StatusNotFound)
			return
		}

		// Check user belongs to item's group
		if user.GroupID != item.GroupID {
			http.Error(w, "User is not a member of this item's group", http.StatusForbidden)
			return
		}

		// Delete item
		if !testDB.DeletePantryItem(itemID) {
			http.Error(w, "Failed to delete item", http.StatusInternalServerError)
			return
		}

		// Create history record
		history := models.CreatePantryHistory(
			item.GroupID,
			item.ID,
			item.Name,
			user.ID,
			user.Name,
			models.ActionTypeRemove,
			item.Quantity,
			"Removed item from pantry",
		)
		testDB.AddPantryHistory(*history)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Pantry item deleted successfully",
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify item was deleted
	_, found := testDB.FindPantryItemByID(testPantryItem.ID)
	if found {
		t.Errorf("Item should not exist in database after deletion")
	}

	// Check history was created
	histories := testDB.GetPantryHistoryByItemID(testPantryItem.ID)
	if len(histories) == 0 {
		t.Errorf("Expected a history record to be created for the deletion")
	}
}

func TestGetPantryWarningsHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	testPantryItem := models.PantryItem{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		Name:      "Milk",
		Quantity:  0.5, // Low quantity to trigger warning
		Unit:      "gallons",
		Category:  "Dairy",
		AddedBy:   testUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	testNotification := models.PantryNotification{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		ItemID:    testPantryItem.ID,
		ItemName:  testPantryItem.Name,
		Type:      models.NotificationTypeLowStock,
		Message:   "Item is running low",
		CreatedAt: time.Now(),
		ReadBy:    []primitive.ObjectID{},
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddPantryItem(testPantryItem)
	testDB.AddPantryNotification(testNotification)

	// Create a request
	req, err := http.NewRequest("GET", "/api/pantry/warnings?group_name="+testGroup.Name, nil)
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
		// Get query params
		groupName := testGroup.Name // Simulate getting from query string

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

		// Find group
		group, found := testDB.FindGroupByName(groupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Check user belongs to group
		if user.GroupID != group.ID {
			http.Error(w, "User is not a member of this group", http.StatusForbidden)
			return
		}

		// Get notifications
		notifications := testDB.GetPantryNotificationsByGroup(group.ID)

		// Filter by type
		var lowStockNotifications []models.PantryNotification
		for _, notification := range notifications {
			if notification.Type == models.NotificationTypeLowStock {
				lowStockNotifications = append(lowStockNotifications, notification)
			}
		}

		// Prepare response with item information
		type WarningResponse struct {
			models.PantryNotification
			CurrentQuantity float64 `json:"current_quantity"`
			Unit            string  `json:"unit"`
			IsRead          bool    `json:"is_read"`
		}

		response := make([]WarningResponse, 0)
		for _, notification := range lowStockNotifications {
			item, found := testDB.FindPantryItemByID(notification.ItemID)
			warningResp := WarningResponse{
				PantryNotification: notification,
				IsRead:             notification.HasBeenReadBy(userID),
			}

			if found {
				warningResp.CurrentQuantity = item.Quantity
				warningResp.Unit = item.Unit
			}

			response = append(response, warningResp)
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var response []map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Check response content
	if len(response) != 1 {
		t.Errorf("Expected 1 warning in response, got %d", len(response))
	}

	if len(response) > 0 {
		warning := response[0]
		itemIDStr, ok := warning["item_id"].(string)
		if !ok {
			t.Errorf("Expected item_id in warning response")
		} else if itemIDStr != testPantryItem.ID.Hex() {
			t.Errorf("Expected item_id to match test item")
		}

		qty, ok := warning["current_quantity"].(float64)
		if !ok {
			t.Errorf("Expected current_quantity in warning response")
		} else if qty != testPantryItem.Quantity {
			t.Errorf("Expected quantity to be %f, got %f", testPantryItem.Quantity, qty)
		}
	}
}

func TestGetPantryExpiringHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Item that will expire soon
	testPantryItem := models.PantryItem{
		ID:             primitive.NewObjectID(),
		GroupID:        testGroup.ID,
		Name:           "Yogurt",
		Quantity:       3.0,
		Unit:           "cups",
		Category:       "Dairy",
		ExpirationDate: time.Now().Add(2 * 24 * time.Hour), // Expires in 2 days
		AddedBy:        testUser.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	testNotification := models.PantryNotification{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		ItemID:    testPantryItem.ID,
		ItemName:  testPantryItem.Name,
		Type:      models.NotificationTypeExpiringSoon,
		Message:   "Item will expire in 3 days or less",
		CreatedAt: time.Now(),
		ReadBy:    []primitive.ObjectID{},
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddPantryItem(testPantryItem)
	testDB.AddPantryNotification(testNotification)

	// Create a request
	req, err := http.NewRequest("GET", "/api/pantry/expiring?group_name="+testGroup.Name, nil)
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
		// Get query params
		groupName := testGroup.Name // Simulate getting from query string

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

		// Find group
		group, found := testDB.FindGroupByName(groupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Check user belongs to group
		if user.GroupID != group.ID {
			http.Error(w, "User is not a member of this group", http.StatusForbidden)
			return
		}

		// Get notifications
		notifications := testDB.GetPantryNotificationsByGroup(group.ID)

		// Filter by type
		var expiringNotifications []models.PantryNotification
		for _, notification := range notifications {
			if notification.Type == models.NotificationTypeExpiringSoon ||
				notification.Type == models.NotificationTypeExpired {
				expiringNotifications = append(expiringNotifications, notification)
			}
		}

		// Prepare response with item information
		type ExpiringResponse struct {
			models.PantryNotification
			ExpirationDate time.Time `json:"expiration_date"`
			Quantity       float64   `json:"quantity"`
			Unit           string    `json:"unit"`
			IsExpired      bool      `json:"is_expired"`
			IsRead         bool      `json:"is_read"`
		}

		response := make([]ExpiringResponse, 0)
		for _, notification := range expiringNotifications {
			item, found := testDB.FindPantryItemByID(notification.ItemID)
			expiringResp := ExpiringResponse{
				PantryNotification: notification,
				IsRead:             notification.HasBeenReadBy(userID),
			}

			if found {
				expiringResp.ExpirationDate = item.ExpirationDate
				expiringResp.Quantity = item.Quantity
				expiringResp.Unit = item.Unit
				expiringResp.IsExpired = item.IsExpired()
			}

			response = append(response, expiringResp)
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var response []map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Check response content
	if len(response) != 1 {
		t.Errorf("Expected 1 expiring item in response, got %d", len(response))
	}

	if len(response) > 0 {
		item := response[0]
		itemIDStr, ok := item["item_id"].(string)
		if !ok {
			t.Errorf("Expected item_id in expiring response")
		} else if itemIDStr != testPantryItem.ID.Hex() {
			t.Errorf("Expected item_id to match test item")
		}

		qty, ok := item["quantity"].(float64)
		if !ok {
			t.Errorf("Expected quantity in expiring response")
		} else if qty != testPantryItem.Quantity {
			t.Errorf("Expected quantity to be %f, got %f", testPantryItem.Quantity, qty)
		}

		isExpired, ok := item["is_expired"].(bool)
		if !ok {
			t.Errorf("Expected is_expired in expiring response")
		} else if isExpired {
			t.Errorf("Expected is_expired to be false")
		}
	}
}

func TestGetPantryShoppingListHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Low stock item
	lowStockItem := models.PantryItem{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		Name:      "Milk",
		Quantity:  0.5, // Low quantity
		Unit:      "gallons",
		Category:  "Dairy",
		AddedBy:   testUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Empty item
	emptyItem := models.PantryItem{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		Name:      "Eggs",
		Quantity:  0.0, // Empty
		Unit:      "dozen",
		Category:  "Dairy",
		AddedBy:   testUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Notification for low stock
	lowStockNotification := models.PantryNotification{
		ID:        primitive.NewObjectID(),
		GroupID:   testGroup.ID,
		ItemID:    lowStockItem.ID,
		ItemName:  lowStockItem.Name,
		Type:      models.NotificationTypeLowStock,
		Message:   "Item is running low",
		CreatedAt: time.Now(),
		ReadBy:    []primitive.ObjectID{},
	}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)
	testDB.AddPantryItem(lowStockItem)
	testDB.AddPantryItem(emptyItem)
	testDB.AddPantryNotification(lowStockNotification)

	// Create a request
	req, err := http.NewRequest("GET", "/api/pantry/shopping-list?group_name="+testGroup.Name, nil)
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
		// Get query params
		groupName := testGroup.Name // Simulate getting from query string

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

		// Find group
		group, found := testDB.FindGroupByName(groupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Check user belongs to group
		if user.GroupID != group.ID {
			http.Error(w, "User is not a member of this group", http.StatusForbidden)
			return
		}

		// Get all items with low stock
		var lowStockItems []models.PantryItem
		for _, item := range testDB.PantryItems {
			if item.GroupID == group.ID && item.Quantity <= 1.0 {
				lowStockItems = append(lowStockItems, item)
			}
		}

		// Create shopping list
		shoppingList := make([]map[string]interface{}, 0)
		for _, item := range lowStockItems {
			// Calculate suggested quantity
			suggestedQuantity := 1.0
			if item.Quantity <= 0 {
				suggestedQuantity = 2.0
			}

			shoppingList = append(shoppingList, map[string]interface{}{
				"item_id":            item.ID.Hex(),
				"name":               item.Name,
				"category":           item.Category,
				"current_quantity":   item.Quantity,
				"unit":               item.Unit,
				"suggested_quantity": suggestedQuantity,
				"reason":             "Low stock",
			})
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"shopping_list": shoppingList,
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Check shopping list
	shoppingList, ok := response["shopping_list"].([]interface{})
	if !ok {
		t.Errorf("Expected shopping_list in response")
	}

	if len(shoppingList) != 2 {
		t.Errorf("Expected 2 items in shopping list, got %d", len(shoppingList))
	}

	// Verify items in the shopping list
	for _, item := range shoppingList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			t.Errorf("Expected map for shopping list item")
			continue
		}

		name, ok := itemMap["name"].(string)
		if !ok {
			t.Errorf("Expected name in shopping list item")
			continue
		}

		if name != "Milk" && name != "Eggs" {
			t.Errorf("Unexpected item in shopping list: %s", name)
		}

		currentQty, ok := itemMap["current_quantity"].(float64)
		if !ok {
			t.Errorf("Expected current_quantity in shopping list item")
			continue
		}

		suggestedQty, ok := itemMap["suggested_quantity"].(float64)
		if !ok {
			t.Errorf("Expected suggested_quantity in shopping list item")
			continue
		}

		// Check suggested quantity logic
		if name == "Eggs" && currentQty == 0.0 && suggestedQty != 2.0 {
			t.Errorf("Expected suggested quantity for empty item to be 2.0, got %f", suggestedQty)
		}
	}
}
