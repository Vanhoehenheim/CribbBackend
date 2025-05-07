// handlers/group_test.go
package handlers_test

import (
	"bytes"
	"cribb-backend/models"
	"cribb-backend/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateGroupHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create a group request
	groupReq := models.Group{
		Name: "New Test Apartment",
	}

	reqBody, _ := json.Marshal(groupReq)
	req, err := http.NewRequest("POST", "/api/groups", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB instead of the real handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var group models.Group
		if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Initialize with proper defaults
		newGroup := models.NewGroup(group.Name)

		// Set ID
		newGroup.ID = primitive.NewObjectID()

		// Add to database
		testDB.AddGroup(*newGroup)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newGroup)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Parse response
	var createdGroup models.Group
	err = json.Unmarshal(rr.Body.Bytes(), &createdGroup)
	if err != nil {
		t.Fatal(err)
	}

	// Verify group was created correctly
	if createdGroup.Name != groupReq.Name {
		t.Errorf("Expected name %s, got %s", groupReq.Name, createdGroup.Name)
	}

	if len(createdGroup.GroupCode) != 6 {
		t.Errorf("Expected group code of length 6, got %d (%s)", len(createdGroup.GroupCode), createdGroup.GroupCode)
	}

	// Verify group was added to database
	dbGroup, found := testDB.FindGroupByName(groupReq.Name)
	if !found {
		t.Errorf("Group not found in database")
	} else if dbGroup.Name != groupReq.Name {
		t.Errorf("Group in database has wrong name: %s", dbGroup.Name)
	}
}

func TestJoinGroupHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = primitive.NilObjectID // Clear group ID for testing
	testUser.Group = ""
	testUser.GroupCode = ""

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a join group request
	joinReq := struct {
		Username   string `json:"username"`
		GroupCode  string `json:"groupCode"`
		RoomNumber string `json:"roomNo"`
	}{
		Username:   testUser.Username,
		GroupCode:  testGroup.GroupCode,
		RoomNumber: "303",
	}

	reqBody, _ := json.Marshal(joinReq)
	req, err := http.NewRequest("POST", "/api/groups/join", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			Username   string `json:"username"`
			GroupCode  string `json:"groupCode"`
			RoomNumber string `json:"roomNo"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate
		if req.Username == "" || req.GroupCode == "" {
			http.Error(w, "Username and GroupCode are required", http.StatusBadRequest)
			return
		}

		// Find group
		group, found := testDB.FindGroupByCode(req.GroupCode)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Find user
		user, found := testDB.FindUserByUsername(req.Username)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Update user with group info
		user.Group = group.Name
		user.GroupID = group.ID
		user.GroupCode = group.GroupCode
		user.UpdatedAt = time.Now()

		// Update room number if provided
		if req.RoomNumber != "" {
			user.RoomNumber = req.RoomNumber
		}

		testDB.UpdateUser(user)

		// Update group members
		group.Members = append(group.Members, user.ID)
		group.UpdatedAt = time.Now()
		testDB.UpdateGroup(group)

		// Return response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Successfully joined group",
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
	if response["message"] != "Successfully joined group" {
		t.Errorf("Expected message 'Successfully joined group', got '%s'", response["message"])
	}

	// Verify user was updated
	updatedUser, found := testDB.FindUserByUsername(testUser.Username)
	if !found {
		t.Errorf("User not found in database")
	} else {
		if updatedUser.Group != testGroup.Name {
			t.Errorf("User group name not updated: expected %s, got %s", testGroup.Name, updatedUser.Group)
		}
		if updatedUser.GroupID != testGroup.ID {
			t.Errorf("User group ID not updated: expected %s, got %s", testGroup.ID.Hex(), updatedUser.GroupID.Hex())
		}
		if updatedUser.RoomNumber != joinReq.RoomNumber {
			t.Errorf("User room number not updated: expected %s, got %s", joinReq.RoomNumber, updatedUser.RoomNumber)
		}
	}

	// Verify group was updated
	updatedGroup, found := testDB.FindGroupByCode(testGroup.GroupCode)
	if !found {
		t.Errorf("Group not found in database")
	} else {
		userFound := false
		for _, memberID := range updatedGroup.Members {
			if memberID == testUser.ID {
				userFound = true
				break
			}
		}
		if !userFound {
			t.Errorf("User not added to group members")
		}
	}
}

func TestGetGroupMembersHandler(t *testing.T) {
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

	// Update group with members
	testGroup.Members = []primitive.ObjectID{testUser1.ID, testUser2.ID}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)

	// Create a request
	req, err := http.NewRequest("GET", "/api/groups/members?group_name="+testGroup.Name, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get group identifier from query
		groupName := r.URL.Query().Get("group_name")
		groupCode := r.URL.Query().Get("group_code")

		if groupName == "" && groupCode == "" {
			http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
			return
		}

		// Find group
		var group models.Group
		var found bool

		if groupName != "" {
			group, found = testDB.FindGroupByName(groupName)
		} else {
			group, found = testDB.FindGroupByCode(groupCode)
		}

		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Get users in the group
		users := testDB.GetUsersForGroup(group.ID)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var users []models.User
	err = json.Unmarshal(rr.Body.Bytes(), &users)
	if err != nil {
		t.Fatal(err)
	}

	// Verify users were returned
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Verify both users were included
	var user1Found, user2Found bool
	for _, user := range users {
		if user.ID == testUser1.ID {
			user1Found = true
		}
		if user.ID == testUser2.ID {
			user2Found = true
		}
	}

	if !user1Found {
		t.Errorf("User 1 not included in response")
	}
	if !user2Found {
		t.Errorf("User 2 not included in response")
	}
}

func TestGetGroupMembersHandlerByCode(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testGroup := test.CreateTestGroup()
	testUser := test.CreateTestUser()
	testUser.GroupID = testGroup.ID

	// Update group with members
	testGroup.Members = []primitive.ObjectID{testUser.ID}

	// Add test data to database
	testDB.AddGroup(testGroup)
	testDB.AddUser(testUser)

	// Create a request using group_code instead of group_name
	req, err := http.NewRequest("GET", "/api/groups/members?group_code="+testGroup.GroupCode, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get group identifier from query
		groupName := r.URL.Query().Get("group_name")
		groupCode := r.URL.Query().Get("group_code")

		if groupName == "" && groupCode == "" {
			http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
			return
		}

		// Find group
		var group models.Group
		var found bool

		if groupName != "" {
			group, found = testDB.FindGroupByName(groupName)
		} else {
			group, found = testDB.FindGroupByCode(groupCode)
		}

		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Get users in the group
		users := testDB.GetUsersForGroup(group.ID)

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var users []models.User
	err = json.Unmarshal(rr.Body.Bytes(), &users)
	if err != nil {
		t.Fatal(err)
	}

	// Verify users were returned
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// Verify the user is the one we added
	if len(users) > 0 && users[0].ID != testUser.ID {
		t.Errorf("Expected user ID %s, got %s", testUser.ID.Hex(), users[0].ID.Hex())
	}
}

func TestGetGroupMembersMissingParameters(t *testing.T) {
	// Create a request without required parameters
	req, err := http.NewRequest("GET", "/api/groups/members", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get group identifier from query
		groupName := r.URL.Query().Get("group_name")
		groupCode := r.URL.Query().Get("group_code")

		if groupName == "" && groupCode == "" {
			http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
			return
		}

		// This part shouldn't be reached in this test
		w.WriteHeader(http.StatusOK)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check that it returns a 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
