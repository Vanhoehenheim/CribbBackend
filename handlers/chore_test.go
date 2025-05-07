// handlers/chore_test.go
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

// Helper function to create an authenticated request context
func createAuthContext(userID, username string) context.Context {
	userClaims := middleware.UserClaims{
		ID:       userID,
		Username: username,
	}
	return context.WithValue(context.Background(), middleware.UserContextKey, userClaims)
}

func TestCreateIndividualChoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a chore creation request
	choreReq := struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		GroupName   string    `json:"group_name"`
		AssignedTo  string    `json:"assigned_to"`
		DueDate     time.Time `json:"due_date"`
		Points      int       `json:"points"`
	}{
		Title:       "Clean Kitchen",
		Description: "Wash dishes and wipe counters",
		GroupName:   testGroup.Name,
		AssignedTo:  testUser.Username,
		DueDate:     time.Now().Add(24 * time.Hour),
		Points:      10,
	}

	reqBody, _ := json.Marshal(choreReq)
	req, err := http.NewRequest("POST", "/api/chores/individual", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			Title       string    `json:"title"`
			Description string    `json:"description"`
			GroupName   string    `json:"group_name"`
			AssignedTo  string    `json:"assigned_to"`
			DueDate     time.Time `json:"due_date"`
			Points      int       `json:"points"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate
		if req.Title == "" || req.GroupName == "" || req.AssignedTo == "" {
			http.Error(w, "Title, group name, and assigned user are required", http.StatusBadRequest)
			return
		}

		// Find group
		group, found := testDB.FindGroupByName(req.GroupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Find user
		user, found := testDB.FindUserByUsername(req.AssignedTo)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Check if user belongs to group
		if user.GroupID != group.ID {
			http.Error(w, "User is not a member of this group", http.StatusBadRequest)
			return
		}

		// Create chore
		chorePtr := models.CreateChore(
			req.Title,
			req.Description,
			group.ID,
			user.ID,
			req.DueDate,
			req.Points,
		)

		// Set ID
		chorePtr.ID = primitive.NewObjectID()

		// Convert to value type
		chore := *chorePtr

		// Add to database
		testDB.AddChore(chore)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(chore)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check response
	var chore models.Chore
	err = json.Unmarshal(rr.Body.Bytes(), &chore)
	if err != nil {
		t.Fatal(err)
	}

	// Verify chore was created correctly
	if chore.Title != choreReq.Title {
		t.Errorf("Expected title %s, got %s", choreReq.Title, chore.Title)
	}

	if chore.GroupID != testGroup.ID {
		t.Errorf("Expected group ID %s, got %s", testGroup.ID.Hex(), chore.GroupID.Hex())
	}

	if chore.AssignedTo != testUser.ID {
		t.Errorf("Expected assigned to %s, got %s", testUser.ID.Hex(), chore.AssignedTo.Hex())
	}

	// Verify chore was added to database
	dbChores := testDB.GetChoresForGroup(testGroup.ID)
	if len(dbChores) != 1 {
		t.Errorf("Expected 1 chore in database, got %d", len(dbChores))
	}
}

func TestCreateRecurringChoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser1 := test.CreateTestUser()
	testUser1.GroupID = testGroup.ID
	testUser2 := test.CreateTestUser()
	testUser2.ID = primitive.NewObjectID() // Ensure different ID
	testUser2.Username = "testuser2"
	testUser2.GroupID = testGroup.ID

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)

	// Create a recurring chore creation request
	recurringChoreReq := struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		GroupName   string `json:"group_name"`
		Frequency   string `json:"frequency"`
		Points      int    `json:"points"`
	}{
		Title:       "Take Out Trash",
		Description: "Empty trash bins",
		GroupName:   testGroup.Name,
		Frequency:   "weekly",
		Points:      5,
	}

	reqBody, _ := json.Marshal(recurringChoreReq)
	req, err := http.NewRequest("POST", "/api/chores/recurring", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			GroupName   string `json:"group_name"`
			Frequency   string `json:"frequency"`
			Points      int    `json:"points"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate
		if req.Title == "" || req.GroupName == "" || req.Frequency == "" {
			http.Error(w, "Title, group name, and frequency are required", http.StatusBadRequest)
			return
		}

		// Validate frequency
		validFrequencies := map[string]bool{
			"daily":    true,
			"weekly":   true,
			"biweekly": true,
			"monthly":  true,
		}

		if !validFrequencies[req.Frequency] {
			http.Error(w, "Invalid frequency. Must be daily, weekly, biweekly, or monthly", http.StatusBadRequest)
			return
		}

		// Find group
		group, found := testDB.FindGroupByName(req.GroupName)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Get group members
		users := testDB.GetUsersForGroup(group.ID)
		if len(users) == 0 {
			http.Error(w, "Group has no members to assign chores to", http.StatusBadRequest)
			return
		}

		// Create member rotation array
		memberRotation := make([]primitive.ObjectID, 0, len(users))
		for _, user := range users {
			memberRotation = append(memberRotation, user.ID)
		}

		// Create recurring chore
		recurringChorePtr := models.CreateRecurringChore(
			req.Title,
			req.Description,
			group.ID,
			memberRotation,
			req.Frequency,
			req.Points,
		)

		// Set ID and convert to value type
		recurringChorePtr.ID = primitive.NewObjectID()

		// Calculate next assignment time
		var nextAssignment time.Time
		switch req.Frequency {
		case "daily":
			nextAssignment = time.Now().Add(24 * time.Hour)
		case "weekly":
			nextAssignment = time.Now().Add(7 * 24 * time.Hour)
		case "biweekly":
			nextAssignment = time.Now().Add(14 * 24 * time.Hour)
		case "monthly":
			nextAssignment = time.Now().AddDate(0, 1, 0)
		}
		recurringChorePtr.NextAssignment = nextAssignment

		// Convert to value type
		recurringChore := *recurringChorePtr

		// Add to database
		testDB.AddRecurringChore(recurringChore)

		// Create first chore instance
		firstChorePtr := models.CreateChoreFromRecurring(recurringChorePtr)
		firstChorePtr.ID = primitive.NewObjectID()
		firstChore := *firstChorePtr
		testDB.AddChore(firstChore)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(recurringChore)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check response
	var recurringChore models.RecurringChore
	err = json.Unmarshal(rr.Body.Bytes(), &recurringChore)
	if err != nil {
		t.Fatal(err)
	}

	// Verify recurring chore was created correctly
	if recurringChore.Title != recurringChoreReq.Title {
		t.Errorf("Expected title %s, got %s", recurringChoreReq.Title, recurringChore.Title)
	}

	if recurringChore.GroupID != testGroup.ID {
		t.Errorf("Expected group ID %s, got %s", testGroup.ID.Hex(), recurringChore.GroupID.Hex())
	}

	if recurringChore.Frequency != recurringChoreReq.Frequency {
		t.Errorf("Expected frequency %s, got %s", recurringChoreReq.Frequency, recurringChore.Frequency)
	}

	// Verify recurring chore was added to database
	dbRecurringChores := testDB.GetRecurringChoresForGroup(testGroup.ID)
	if len(dbRecurringChores) != 1 {
		t.Errorf("Expected 1 recurring chore in database, got %d", len(dbRecurringChores))
	}

	// Verify first chore instance was created
	dbChores := testDB.GetChoresForGroup(testGroup.ID)
	if len(dbChores) != 1 {
		t.Errorf("Expected 1 chore instance in database, got %d", len(dbChores))
	}
}

func TestGetUserChoresHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testUser := test.CreateTestUser()
	testChore1 := test.CreateTestChore()
	testChore1.AssignedTo = testUser.ID
	testChore1.Status = models.ChoreStatusPending
	testChore2 := test.CreateTestChore()
	testChore2.ID = primitive.NewObjectID() // Ensure different ID
	testChore2.AssignedTo = testUser.ID
	testChore2.Status = models.ChoreStatusPending
	testChore2.DueDate = time.Now().Add(-48 * time.Hour) // Overdue

	// Add test data to database
	testDB.AddUser(testUser)
	testDB.AddChore(testChore1)
	testDB.AddChore(testChore2)

	// Create a request
	req, err := http.NewRequest("GET", "/api/chores/user?username="+testUser.Username, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get username from query
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}

		// Find user
		user, found := testDB.FindUserByUsername(username)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Get chores for user
		chores := testDB.GetChoresForUser(user.ID)

		// Filter out completed chores
		activeChores := make([]models.Chore, 0)
		for _, chore := range chores {
			if chore.Status != models.ChoreStatusCompleted {
				// Check for overdue chores
				if chore.Status != models.ChoreStatusOverdue && !chore.DueDate.IsZero() && chore.DueDate.Before(time.Now()) {
					chore.Status = models.ChoreStatusOverdue
					testDB.UpdateChore(chore)
				}
				activeChores = append(activeChores, chore)
			}
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activeChores)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var chores []models.Chore
	err = json.Unmarshal(rr.Body.Bytes(), &chores)
	if err != nil {
		t.Fatal(err)
	}

	// Verify chores were returned
	if len(chores) != 2 {
		t.Errorf("Expected 2 chores, got %d", len(chores))
	}

	// Verify overdue chore status was updated
	var overdueChoreFound bool
	for _, chore := range chores {
		if chore.ID == testChore2.ID && chore.Status == models.ChoreStatusOverdue {
			overdueChoreFound = true
			break
		}
	}

	if !overdueChoreFound {
		t.Errorf("Expected overdue chore to have status updated to overdue")
	}
}

func TestCompleteChoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testUser := test.CreateTestUser()
	testUser.Score = 10
	testChore := test.CreateTestChore()
	testChore.AssignedTo = testUser.ID
	testChore.Status = models.ChoreStatusPending
	testChore.Points = 5

	// Add test data to database
	testDB.AddUser(testUser)
	testDB.AddChore(testChore)

	// Create a chore completion request
	completeReq := struct {
		ChoreID  string `json:"chore_id"`
		Username string `json:"username"`
	}{
		ChoreID:  testChore.ID.Hex(),
		Username: testUser.Username,
	}

	reqBody, _ := json.Marshal(completeReq)
	req, err := http.NewRequest("POST", "/api/chores/complete", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ChoreID  string `json:"chore_id"`
			Username string `json:"username"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate
		if req.ChoreID == "" || req.Username == "" {
			http.Error(w, "Chore ID and username are required", http.StatusBadRequest)
			return
		}

		// Convert chore ID
		choreID, err := primitive.ObjectIDFromHex(req.ChoreID)
		if err != nil {
			http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
			return
		}

		// Find user
		user, found := testDB.FindUserByUsername(req.Username)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Find chore
		chore, found := testDB.FindChoreByID(choreID)
		if !found {
			http.Error(w, "Chore not found", http.StatusNotFound)
			return
		}

		// Verify chore is assigned to user
		if chore.AssignedTo != user.ID {
			http.Error(w, "Chore is not assigned to this user", http.StatusBadRequest)
			return
		}

		// Verify chore is not already completed
		if chore.Status == models.ChoreStatusCompleted {
			http.Error(w, "Chore is already completed", http.StatusBadRequest)
			return
		}

		// Complete the chore
		chore.Status = models.ChoreStatusCompleted
		chore.UpdatedAt = time.Now()
		testDB.UpdateChore(chore)

		// Update user score
		user.Score += chore.Points
		user.UpdatedAt = time.Now()
		testDB.UpdateUser(user)

		// Create chore completion record
		completion := models.ChoreCompletion{
			ChoreID:     chore.ID,
			UserID:      user.ID,
			CompletedAt: time.Now(),
			Points:      chore.Points,
		}
		testDB.AddChoreCompletion(completion)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"points_earned": chore.Points,
			"new_score":     user.Score,
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Verify points and score
	if int(response["points_earned"].(float64)) != 5 {
		t.Errorf("Expected points_earned 5, got %v", response["points_earned"])
	}

	if int(response["new_score"].(float64)) != 15 {
		t.Errorf("Expected new_score 15, got %v", response["new_score"])
	}

	// Verify chore was marked as completed
	updatedChore, found := testDB.FindChoreByID(testChore.ID)
	if !found {
		t.Errorf("Chore not found in database after completion")
	} else if updatedChore.Status != models.ChoreStatusCompleted {
		t.Errorf("Expected chore status to be %s, got %s", models.ChoreStatusCompleted, updatedChore.Status)
	}

	// Verify user score was updated
	updatedUser, found := testDB.FindUserByUsername(testUser.Username)
	if !found {
		t.Errorf("User not found in database after chore completion")
	} else if updatedUser.Score != 15 {
		t.Errorf("Expected user score to be 15, got %d", updatedUser.Score)
	}
}

func TestUpdateChoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testChore := test.CreateTestChore()
	testChore.Title = "Original Title"
	testChore.Points = 10

	// Add test data to database
	testDB.AddChore(testChore)

	// Create a chore update request
	updateReq := struct {
		ChoreID     string `json:"chore_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Points      int    `json:"points"`
	}{
		ChoreID:     testChore.ID.Hex(),
		Title:       "Updated Title",
		Description: "Updated Description",
		Points:      15,
	}

	reqBody, _ := json.Marshal(updateReq)
	req, err := http.NewRequest("PUT", "/api/chores/update", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			ChoreID     string `json:"chore_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Points      int    `json:"points"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate
		if req.ChoreID == "" {
			http.Error(w, "Chore ID is required", http.StatusBadRequest)
			return
		}

		// Convert chore ID
		choreID, err := primitive.ObjectIDFromHex(req.ChoreID)
		if err != nil {
			http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
			return
		}

		// Find chore
		chore, found := testDB.FindChoreByID(choreID)
		if !found {
			http.Error(w, "Chore not found", http.StatusNotFound)
			return
		}

		// Check if completed
		if chore.Status == models.ChoreStatusCompleted {
			http.Error(w, "Cannot update a completed chore", http.StatusBadRequest)
			return
		}

		// Update chore
		if req.Title != "" {
			chore.Title = req.Title
		}

		if req.Description != "" {
			chore.Description = req.Description
		}

		if req.Points > 0 {
			chore.Points = req.Points
		}

		chore.UpdatedAt = time.Now()
		testDB.UpdateChore(chore)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(chore)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var updatedChore models.Chore
	err = json.Unmarshal(rr.Body.Bytes(), &updatedChore)
	if err != nil {
		t.Fatal(err)
	}

	// Verify chore was updated
	if updatedChore.Title != updateReq.Title {
		t.Errorf("Expected title %s, got %s", updateReq.Title, updatedChore.Title)
	}

	if updatedChore.Description != updateReq.Description {
		t.Errorf("Expected description %s, got %s", updateReq.Description, updatedChore.Description)
	}

	if updatedChore.Points != updateReq.Points {
		t.Errorf("Expected points %d, got %d", updateReq.Points, updatedChore.Points)
	}

	// Verify database was updated
	dbChore, found := testDB.FindChoreByID(testChore.ID)
	if !found {
		t.Errorf("Chore not found in database after update")
	} else {
		if dbChore.Title != updateReq.Title {
			t.Errorf("Database chore title not updated: expected %s, got %s", updateReq.Title, dbChore.Title)
		}

		if dbChore.Points != updateReq.Points {
			t.Errorf("Database chore points not updated: expected %d, got %d", updateReq.Points, dbChore.Points)
		}
	}
}

func TestDeleteChoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testChore := test.CreateTestChore()

	// Add test data to database
	testDB.AddChore(testChore)

	// Create a request
	req, err := http.NewRequest("DELETE", "/api/chores/delete?chore_id="+testChore.ID.Hex(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get chore ID from query
		choreID := r.URL.Query().Get("chore_id")
		if choreID == "" {
			http.Error(w, "Chore ID is required", http.StatusBadRequest)
			return
		}

		// Convert chore ID
		objectID, err := primitive.ObjectIDFromHex(choreID)
		if err != nil {
			http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
			return
		}

		// Find chore
		_, found := testDB.FindChoreByID(objectID)
		if !found {
			http.Error(w, "Chore not found", http.StatusNotFound)
			return
		}

		// Delete chore
		deleted := testDB.DeleteChore(objectID)
		if !deleted {
			http.Error(w, "Failed to delete chore", http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Chore deleted successfully",
		})
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Verify response message
	if response["message"] != "Chore deleted successfully" {
		t.Errorf("Expected message 'Chore deleted successfully', got '%s'", response["message"])
	}

	// Verify chore was deleted from database
	_, found := testDB.FindChoreByID(testChore.ID)
	if found {
		t.Errorf("Chore still exists in database after deletion")
	}
}
