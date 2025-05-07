// handlers/chore.go
package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateIndividualChoreHandler creates a new individual chore
func CreateIndividualChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		GroupName   string    `json:"group_name"`
		AssignedTo  string    `json:"assigned_to"` // Username of user to assign
		DueDate     time.Time `json:"due_date"`
		Points      int       `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.Title == "" || request.GroupName == "" || request.AssignedTo == "" {
		http.Error(w, "Title, group name, and assigned user are required", http.StatusBadRequest)
		return
	}

	if request.Points < 1 {
		request.Points = 1 // Default points if not provided or invalid
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

	// Find the user
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"username": request.AssignedTo},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Check if user belongs to this group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusBadRequest)
		return
	}

	// Create the chore
	chore := models.CreateChore(
		request.Title,
		request.Description,
		group.ID,
		user.ID,
		request.DueDate,
		request.Points,
	)

	// Insert the chore
	result, err := config.DB.Collection("chores").InsertOne(context.Background(), chore)
	if err != nil {
		log.Printf("Chore creation error: %v", err)
		http.Error(w, "Failed to create chore", http.StatusInternalServerError)
		return
	}

	// Set the inserted ID
	chore.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chore)
}

// GetUserChoresHandler retrieves all chores assigned to a user
// CreateRecurringChoreHandler creates a new recurring chore
func CreateRecurringChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		GroupName   string `json:"group_name"`
		Frequency   string `json:"frequency"` // daily, weekly, biweekly, monthly
		Points      int    `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.Title == "" || request.GroupName == "" || request.Frequency == "" {
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

	if !validFrequencies[request.Frequency] {
		http.Error(w, "Invalid frequency. Must be daily, weekly, biweekly, or monthly", http.StatusBadRequest)
		return
	}

	if request.Points < 1 {
		request.Points = 1 // Default points if not provided or invalid
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

	// Fetch group members for rotation
	cursor, err := config.DB.Collection("users").Find(
		context.Background(),
		bson.M{"group_id": group.ID},
	)
	if err != nil {
		http.Error(w, "Failed to fetch group members", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err = cursor.All(context.Background(), &users); err != nil {
		http.Error(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	if len(users) == 0 {
		http.Error(w, "Group has no members to assign chores to", http.StatusBadRequest)
		return
	}

	// Create member rotation array
	memberRotation := make([]primitive.ObjectID, 0, len(users))
	for _, user := range users {
		memberRotation = append(memberRotation, user.ID)
	}

	// Create the recurring chore
	recurringChore := models.CreateRecurringChore(
		request.Title,
		request.Description,
		group.ID,
		memberRotation,
		request.Frequency,
		request.Points,
	)

	// Calculate next assignment time based on frequency
	var nextAssignment time.Time
	switch request.Frequency {
	case "daily":
		nextAssignment = time.Now().Add(24 * time.Hour)
	case "weekly":
		nextAssignment = time.Now().Add(7 * 24 * time.Hour)
	case "biweekly":
		nextAssignment = time.Now().Add(14 * 24 * time.Hour)
	case "monthly":
		nextAssignment = time.Now().AddDate(0, 1, 0)
	}
	recurringChore.NextAssignment = nextAssignment

	// Insert the recurring chore
	result, err := config.DB.Collection("recurring_chores").InsertOne(context.Background(), recurringChore)
	if err != nil {
		log.Printf("Recurring chore creation error: %v", err)
		http.Error(w, "Failed to create recurring chore", http.StatusInternalServerError)
		return
	}

	// Set the inserted ID
	recurringChore.ID = result.InsertedID.(primitive.ObjectID)

	// Create the first instance of this recurring chore
	firstChore := models.CreateChoreFromRecurring(recurringChore)

	// Insert the first chore instance
	_, err = config.DB.Collection("chores").InsertOne(context.Background(), firstChore)
	if err != nil {
		log.Printf("Failed to create first chore instance: %v", err)
		// Continue anyway since the recurring definition was created successfully
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recurringChore)
}

func GetUserChoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Find the user
	var user models.User
	err := config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"username": username},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Get all active chores for the user
	cursor, err := config.DB.Collection("chores").Find(
		context.Background(),
		bson.M{
			"assigned_to": user.ID,
			"status":      bson.M{"$ne": models.ChoreStatusCompleted},
		},
	)
	if err != nil {
		http.Error(w, "Failed to fetch chores", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var chores []models.Chore
	if err = cursor.All(context.Background(), &chores); err != nil {
		http.Error(w, "Failed to decode chores", http.StatusInternalServerError)
		return
	}

	// Check for overdue chores and update their status
	now := time.Now()
	for i, chore := range chores {
		if chore.Status != models.ChoreStatusOverdue && !chore.DueDate.IsZero() && chore.DueDate.Before(now) {
			chores[i].Status = models.ChoreStatusOverdue

			// Update in database
			_, _ = config.DB.Collection("chores").UpdateOne(
				context.Background(),
				bson.M{"_id": chore.ID},
				bson.M{"$set": bson.M{"status": models.ChoreStatusOverdue}},
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chores)
}
