package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cribb-backend/config"
	"cribb-backend/middleware"
	"cribb-backend/models"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	RoomNumber  string `json:"room_number"`         // Changed from roomNo to match User model
	Group       string `json:"group,omitempty"`     // For creating a new group
	GroupCode   string `json:"groupCode,omitempty"` // For joining an existing group
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" || req.Name == "" || req.PhoneNumber == "" || req.RoomNumber == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Ensure either group or groupCode is provided, but not both
	if (req.Group == "" && req.GroupCode == "") || (req.Group != "" && req.GroupCode != "") {
		http.Error(w, "Either group or groupCode must be provided", http.StatusBadRequest)
		return
	}

	// Start a session for transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		http.Error(w, "Failed to start session", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var groupID primitive.ObjectID
	var groupName string
	var groupCode string

	// Execute transaction
	err = session.StartTransaction()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// Handle group creation or joining
		if req.Group != "" {
			// Creating a new group
			newGroup := models.NewGroup(req.Group)
			result, err := config.DB.Collection("groups").InsertOne(sc, newGroup)
			if err != nil {
				if mongo.IsDuplicateKeyError(err) {
					return fmt.Errorf("group name already exists")
				}
				return fmt.Errorf("failed to create group: %v", err)
			}
			groupID = result.InsertedID.(primitive.ObjectID)
			groupName = newGroup.Name
			groupCode = newGroup.GroupCode
		} else {
			// Joining existing group
			var group models.Group
			filter := bson.M{"group_code": req.GroupCode}
			err := config.DB.Collection("groups").FindOne(
				sc,
				filter,
			).Decode(&group)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return fmt.Errorf("group not found")
				}
				return fmt.Errorf("failed to fetch group: %v", err)
			}
			groupID = group.ID
			groupName = group.Name
			groupCode = group.GroupCode
		}

		// Create new user with proper group info
		newUser := models.User{
			ID:          primitive.NewObjectID(),
			Username:    req.Username,
			Password:    string(hashedPassword),
			Name:        req.Name,
			PhoneNumber: req.PhoneNumber,
			RoomNumber:  req.RoomNumber, // Using the correct field name
			Score:       10,
			Group:       groupName,
			GroupID:     groupID,
			GroupCode:   groupCode,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Insert user
		_, err := config.DB.Collection("users").InsertOne(sc, newUser)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return fmt.Errorf("username or phone number already exists")
			}
			return fmt.Errorf("failed to create user: %v", err)
		}

		// Update group with the actual user ID
		_, err = config.DB.Collection("groups").UpdateOne(
			sc,
			bson.M{"_id": groupID},
			bson.M{
				"$push": bson.M{"members": newUser.ID},
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update group with user ID: %v", err)
		}

		// Generate JWT token
		token := GenerateJWTToken(newUser.ID.Hex(), newUser.Username)

		// Split name into first and last name
		nameParts := strings.Split(newUser.Name, " ")
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = strings.Join(nameParts[1:], " ")
		}

		// Prepare response
		response := LoginResponse{
			Success: true,
			Token:   token,
			User: UserData{
				ID:         newUser.ID.Hex(),
				Email:      newUser.Username,
				FirstName:  firstName,
				LastName:   lastName,
				Phone:      newUser.PhoneNumber,
				RoomNumber: newUser.RoomNumber,
				GroupCode:  groupCode,
			},
			Message: "Registration successful",
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		return json.NewEncoder(w).Encode(response)
	})

	if err != nil {
		session.AbortTransaction(context.Background())
		if mongo.IsDuplicateKeyError(err) {
			http.Error(w, "Username, phone number, or group name already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.CommitTransaction(context.Background())
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// GenerateJWTToken creates a new JWT token for the authenticated user
func GenerateJWTToken(userID, username string) string {
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // Token expires in 7 days
	})

	// Sign the token with our secret from config
	tokenString, err := token.SignedString(config.JWTSecret)
	if err != nil {
		return ""
	}

	return tokenString
}

type UserData struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Phone      string `json:"phone"`
	RoomNumber string `json:"roomNo"`
	GroupCode  string `json:"groupCode,omitempty"`
	GroupName  string `json:"groupName,omitempty"`
}

type LoginResponse struct {
	Success bool     `json:"success"`
	Token   string   `json:"token"`
	User    UserData `json:"user"`
	Message string   `json:"message"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Find user by username
	var user models.User
	err := config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"username": req.Username},
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Don't reveal whether username exists or not for security
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to authenticate user", http.StatusInternalServerError)
		return
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		// Password doesn't match
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token := GenerateJWTToken(user.ID.Hex(), user.Username)

	// Split name into first and last name (assuming format is "FirstName LastName")
	nameParts := strings.Split(user.Name, " ")
	firstName := nameParts[0]
	lastName := ""
	if len(nameParts) > 1 {
		lastName = strings.Join(nameParts[1:], " ")
	}

	// Prepare response with user data (excluding password)
	response := LoginResponse{
		Success: true,
		Token:   token,
		User: UserData{
			ID:         user.ID.Hex(),
			Email:      user.Username, // Using username as email
			FirstName:  firstName,
			LastName:   lastName,
			Phone:      user.PhoneNumber,
			RoomNumber: user.RoomNumber,
			GroupCode:  user.GroupCode,
		},
		Message: "Login successful",
	}

	// Return successful login response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
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

	// Find user by ID
	objID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": objID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
		return
	}

	// Split name into first and last name
	nameParts := strings.Split(user.Name, " ")
	firstName := nameParts[0]
	lastName := ""
	if len(nameParts) > 1 {
		lastName = strings.Join(nameParts[1:], " ")
	}

	// Prepare response with group name included
	response := UserData{
		ID:         user.ID.Hex(),
		Email:      user.Username,
		FirstName:  firstName,
		LastName:   lastName,
		Phone:      user.PhoneNumber,
		RoomNumber: user.RoomNumber,
		GroupCode:  user.GroupCode,
		GroupName:  user.Group, // Add the existing group name field
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
