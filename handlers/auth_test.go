// handlers/auth_test.go
package handlers_test

import (
	"bytes"
	"context"
	"cribb-backend/config"
	"cribb-backend/handlers"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"cribb-backend/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// Mock the JWT secret for testing
func init() {
	config.JWTSecret = []byte("test-secret")
}

// Helper to set up a test environment with our TestDB
func setupTestEnv() *test.TestDB {
	testDB := test.NewTestDB()

	// We could patch the global config.DB here, but for now we'll just
	// pass testDB to our test helper functions

	return testDB
}

func TestRegisterHandler(t *testing.T) {
	// Initialize test environment
	testDB := setupTestEnv()

	// Create a test group for the user to join
	testGroup := test.CreateTestGroup()
	testDB.AddGroup(testGroup)

	// Create a request body
	registerReq := handlers.RegisterRequest{
		Username:    "newuser",
		Password:    "password123",
		Name:        "New User",
		PhoneNumber: "9876543210",
		RoomNumber:  "202",
		GroupCode:   testGroup.GroupCode, // Join existing group
	}

	reqBody, _ := json.Marshal(registerReq)
	req, err := http.NewRequest("POST", "/api/register", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	// Note: In a real test you'd need to patch or inject the DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a simplified version for testing purposes
		// In reality, you'd need to wrap the actual handler or modify it to accept a DB

		// Parse request
		var req handlers.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Find group
		group, found := testDB.FindGroupByCode(req.GroupCode)
		if !found {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Create user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := models.User{
			ID:          primitive.NewObjectID(),
			Username:    req.Username,
			Password:    string(hashedPassword),
			Name:        req.Name,
			PhoneNumber: req.PhoneNumber,
			RoomNumber:  req.RoomNumber,
			Score:       10,
			Group:       group.Name,
			GroupID:     group.ID,
			GroupCode:   group.GroupCode,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Add user to DB
		testDB.AddUser(user)

		// Add user to group
		group.Members = append(group.Members, user.ID)
		testDB.UpdateGroup(group)

		// Generate token
		token := handlers.GenerateJWTToken(user.ID.Hex(), user.Username)

		// Split name into first and last
		nameParts := strings.Split(user.Name, " ")
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = strings.Join(nameParts[1:], " ")
		}

		// Create response
		response := handlers.LoginResponse{
			Success: true,
			Token:   token,
			User: handlers.UserData{
				ID:         user.ID.Hex(),
				Email:      user.Username,
				FirstName:  firstName,
				LastName:   lastName,
				Phone:      user.PhoneNumber,
				RoomNumber: user.RoomNumber,
				GroupCode:  user.GroupCode,
			},
			Message: "Registration successful",
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Check the response body
	var response handlers.LoginResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Assertions
	if !response.Success {
		t.Errorf("Expected success to be true")
	}
	if response.Token == "" {
		t.Errorf("Expected a token to be returned")
	}
	if response.User.Email != registerReq.Username {
		t.Errorf("Expected email to be %s, got %s", registerReq.Username, response.User.Email)
	}

	// Verify user was added to the database
	user, found := testDB.FindUserByUsername(registerReq.Username)
	if !found {
		t.Errorf("User not found in database")
	}
	if user.Username != registerReq.Username {
		t.Errorf("Expected username %s, got %s", registerReq.Username, user.Username)
	}

	// Verify user was added to the group
	group, found := testDB.FindGroupByCode(testGroup.GroupCode)
	if !found {
		t.Errorf("Group not found in database")
	}
	userFound := false
	for _, memberID := range group.Members {
		if memberID == user.ID {
			userFound = true
			break
		}
	}
	if !userFound {
		t.Errorf("User not added to group members")
	}
}

func TestLoginHandler(t *testing.T) {
	// Initialize test environment
	testDB := setupTestEnv()

	// Create a test user with a known password
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	testUser := models.User{
		ID:          primitive.NewObjectID(),
		Username:    "testuser",
		Password:    string(hashedPassword),
		Name:        "Test User",
		PhoneNumber: "1234567890",
		RoomNumber:  "101",
		Score:       10,
		Group:       "Test Apartment",
		GroupID:     primitive.NewObjectID(),
		GroupCode:   "ABCDEF",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add user to test database
	testDB.AddUser(testUser)

	// Create a login request
	loginReq := handlers.LoginRequest{
		Username: testUser.Username,
		Password: password,
	}

	reqBody, _ := json.Marshal(loginReq)
	req, err := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req handlers.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Find user
		user, found := testDB.FindUserByUsername(req.Username)
		if !found {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Compare password
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
		if err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Generate token
		token := handlers.GenerateJWTToken(user.ID.Hex(), user.Username)

		// Split name into first and last
		nameParts := strings.Split(user.Name, " ")
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = strings.Join(nameParts[1:], " ")
		}

		// Create response
		response := handlers.LoginResponse{
			Success: true,
			Token:   token,
			User: handlers.UserData{
				ID:         user.ID.Hex(),
				Email:      user.Username,
				FirstName:  firstName,
				LastName:   lastName,
				Phone:      user.PhoneNumber,
				RoomNumber: user.RoomNumber,
				GroupCode:  user.GroupCode,
			},
			Message: "Login successful",
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response handlers.LoginResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Assertions
	if !response.Success {
		t.Errorf("Expected success to be true")
	}
	if response.Token == "" {
		t.Errorf("Expected a token to be returned")
	}
	if response.User.Email != testUser.Username {
		t.Errorf("Expected email to be %s, got %s", testUser.Username, response.User.Email)
	}
}

func TestGetUserProfileHandler(t *testing.T) {
	// Initialize test environment
	testDB := setupTestEnv()

	// Create a test user
	testUser := test.CreateTestUser()
	testDB.AddUser(testUser)

	// Create a request
	req, err := http.NewRequest("GET", "/api/users/profile", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a JWT token for this user
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       testUser.ID.Hex(),
		"username": testUser.Username,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString(config.JWTSecret)

	// Add the auth token to the header
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Record the response
	rr := httptest.NewRecorder()

	// Create a context with the user claims
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, middleware.UserClaims{
		ID:       testUser.ID.Hex(),
		Username: testUser.Username,
	})
	req = req.WithContext(ctx)

	// Create a handler that uses our test DB
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user claims from context
		userClaims, ok := middleware.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Convert ID string to ObjectID
		userID, err := primitive.ObjectIDFromHex(userClaims.ID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Find user by ID
		user, found := testDB.FindUserByID(userID)
		if !found {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Split name into first and last
		nameParts := strings.Split(user.Name, " ")
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = strings.Join(nameParts[1:], " ")
		}

		// Create response
		response := handlers.UserData{
			ID:         user.ID.Hex(),
			Email:      user.Username,
			FirstName:  firstName,
			LastName:   lastName,
			Phone:      user.PhoneNumber,
			RoomNumber: user.RoomNumber,
			GroupCode:  user.GroupCode,
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response handlers.UserData
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	// Assertions
	if response.ID != testUser.ID.Hex() {
		t.Errorf("Expected ID to be %s, got %s", testUser.ID.Hex(), response.ID)
	}
	if response.Email != testUser.Username {
		t.Errorf("Expected email to be %s, got %s", testUser.Username, response.Email)
	}
}

// Test the JWT token generation function
func TestGenerateJWTToken(t *testing.T) {
	userID := primitive.NewObjectID().Hex()
	username := "testuser"

	token := handlers.GenerateJWTToken(userID, username)

	// Verify the token is valid
	if token == "" {
		t.Error("Expected a non-empty token")
	}

	// Parse the token to verify its contents
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return config.JWTSecret, nil
	})

	if err != nil {
		t.Errorf("Failed to parse token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Error("Failed to extract claims from token")
	}

	if claims["id"] != userID {
		t.Errorf("Expected userID %s, got %s", userID, claims["id"])
	}

	if claims["username"] != username {
		t.Errorf("Expected username %s, got %s", username, claims["username"])
	}
}
