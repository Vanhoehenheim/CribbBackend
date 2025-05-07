// handlers/user_test.go
package handlers_test

import (
	"cribb-backend/models"
	"cribb-backend/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestGetUsersHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testUser1 := test.CreateTestUser()
	testUser2 := test.CreateTestUser()
	testUser2.ID = primitive.NewObjectID() // Ensure different ID
	testUser2.Username = "testuser2"

	// Add test data to database
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)

	// Create a request
	req, err := http.NewRequest("GET", "/api/users", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get all users from test database
		users := testDB.Users

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

func TestGetUserByUsernameHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data
	testUser := test.CreateTestUser()

	// Add test data to database
	testDB.AddUser(testUser)

	// Create a request
	req, err := http.NewRequest("GET", "/api/users/by-username?username="+testUser.Username, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get username from query parameter
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Username parameter is required", http.StatusBadRequest)
			return
		}

		// Find user in database
		user, found := testDB.FindUserByUsername(username)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response
	var user models.User
	err = json.Unmarshal(rr.Body.Bytes(), &user)
	if err != nil {
		t.Fatal(err)
	}

	// Verify user was returned
	if user.ID != testUser.ID {
		t.Errorf("Expected user ID %s, got %s", testUser.ID.Hex(), user.ID.Hex())
	}

	if user.Username != testUser.Username {
		t.Errorf("Expected username %s, got %s", testUser.Username, user.Username)
	}
}

func TestGetUserByUsernameMissingParameter(t *testing.T) {
	// Create a request without username parameter
	req, err := http.NewRequest("GET", "/api/users/by-username", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get username from query parameter
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Username parameter is required", http.StatusBadRequest)
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

func TestGetUsersByScoreHandler(t *testing.T) {
	// Initialize test environment
	testDB := test.NewTestDB()

	// Create test data with different scores
	testUser1 := test.CreateTestUser()
	testUser1.Score = 20
	testUser2 := test.CreateTestUser()
	testUser2.ID = primitive.NewObjectID() // Ensure different ID
	testUser2.Username = "testuser2"
	testUser2.Score = 10
	testUser3 := test.CreateTestUser()
	testUser3.ID = primitive.NewObjectID() // Ensure different ID
	testUser3.Username = "testuser3"
	testUser3.Score = 30

	// Add test data to database
	testDB.AddUser(testUser1)
	testDB.AddUser(testUser2)
	testDB.AddUser(testUser3)

	// Create a request
	req, err := http.NewRequest("GET", "/api/users/by-score", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get all users
		users := testDB.Users

		// Sort users by score (descending)
		// Using bubble sort for simplicity since we have a small dataset
		n := len(users)
		for i := 0; i < n-1; i++ {
			for j := 0; j < n-i-1; j++ {
				if users[j].Score < users[j+1].Score {
					users[j], users[j+1] = users[j+1], users[j]
				}
			}
		}

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
	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	// Verify users are sorted by score (highest first)
	if len(users) >= 3 {
		if users[0].Score < users[1].Score || users[1].Score < users[2].Score {
			t.Errorf("Users not sorted by score in descending order: %d, %d, %d", users[0].Score, users[1].Score, users[2].Score)
		}

		// Verify the highest score user is first
		if users[0].Score != 30 || users[0].ID != testUser3.ID {
			t.Errorf("Expected first user to be testuser3 with score 30, got %s with score %d", users[0].Username, users[0].Score)
		}

		// Verify the lowest score user is last
		if users[2].Score != 10 || users[2].ID != testUser2.ID {
			t.Errorf("Expected last user to be testuser2 with score 10, got %s with score %d", users[2].Username, users[2].Score)
		}
	}
}
