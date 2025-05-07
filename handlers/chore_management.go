// handlers/chore_management.go
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

// UpdateChoreHandler handles updating an existing chore
func UpdateChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ChoreID     string    `json:"chore_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		AssignedTo  string    `json:"assigned_to"` // Username of user to assign
		DueDate     time.Time `json:"due_date"`
		Points      int       `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ChoreID == "" {
		http.Error(w, "Chore ID is required", http.StatusBadRequest)
		return
	}

	// Convert chore ID from string to ObjectID
	choreID, err := primitive.ObjectIDFromHex(request.ChoreID)
	if err != nil {
		http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
		return
	}

	// Get existing chore
	var chore models.Chore
	err = config.DB.Collection("chores").FindOne(
		context.Background(),
		bson.M{"_id": choreID},
	).Decode(&chore)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Chore not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch chore", http.StatusInternalServerError)
		}
		return
	}

	// If a chore is already completed, don't allow updates
	if chore.Status == models.ChoreStatusCompleted {
		http.Error(w, "Cannot update a completed chore", http.StatusBadRequest)
		return
	}

	// Prepare update fields
	updateFields := bson.M{
		"updated_at": time.Now(),
	}

	// Only update fields that were provided
	if request.Title != "" {
		updateFields["title"] = request.Title
	}

	if request.Description != "" {
		updateFields["description"] = request.Description
	}

	if !request.DueDate.IsZero() {
		updateFields["due_date"] = request.DueDate
	}

	if request.Points > 0 {
		updateFields["points"] = request.Points
	}

	// If assigned to is changing, need to look up the user ID
	if request.AssignedTo != "" {
		var user models.User
		err = config.DB.Collection("users").FindOne(
			context.Background(),
			bson.M{"username": request.AssignedTo},
		).Decode(&user)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "Assigned user not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
			}
			return
		}

		// Verify the user belongs to the chore's group
		if user.GroupID != chore.GroupID {
			http.Error(w, "User does not belong to this chore's group", http.StatusBadRequest)
			return
		}

		updateFields["assigned_to"] = user.ID
	}

	// Update chore in the database
	result, err := config.DB.Collection("chores").UpdateOne(
		context.Background(),
		bson.M{"_id": choreID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("Failed to update chore: %v", err)
		http.Error(w, "Failed to update chore", http.StatusInternalServerError)
		return
	}

	if result.ModifiedCount == 0 {
		http.Error(w, "No changes were made", http.StatusOK)
		return
	}

	// Get updated chore
	var updatedChore models.Chore
	err = config.DB.Collection("chores").FindOne(
		context.Background(),
		bson.M{"_id": choreID},
	).Decode(&updatedChore)

	if err != nil {
		log.Printf("Failed to fetch updated chore: %v", err)
		// Return success message even though we couldn't fetch the updated document
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Chore updated successfully",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedChore)
}

// DeleteChoreHandler handles deleting a chore
func DeleteChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	choreID := r.URL.Query().Get("chore_id")
	if choreID == "" {
		http.Error(w, "Chore ID is required", http.StatusBadRequest)
		return
	}

	// Convert chore ID from string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(choreID)
	if err != nil {
		http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
		return
	}

	// Get the chore first to check if it's recurring
	var chore models.Chore
	err = config.DB.Collection("chores").FindOne(
		context.Background(),
		bson.M{"_id": objectID},
	).Decode(&chore)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Chore not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch chore", http.StatusInternalServerError)
		}
		return
	}

	// Delete the chore
	result, err := config.DB.Collection("chores").DeleteOne(
		context.Background(),
		bson.M{"_id": objectID},
	)

	if err != nil {
		log.Printf("Failed to delete chore: %v", err)
		http.Error(w, "Failed to delete chore", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Chore not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Chore deleted successfully",
	})
}

// UpdateRecurringChoreHandler handles updating a recurring chore
func UpdateRecurringChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		RecurringChoreID string `json:"recurring_chore_id"`
		Title            string `json:"title"`
		Description      string `json:"description"`
		Frequency        string `json:"frequency"` // daily, weekly, biweekly, monthly
		Points           int    `json:"points"`
		IsActive         *bool  `json:"is_active"` // Pointer to allow nil checks
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.RecurringChoreID == "" {
		http.Error(w, "Recurring chore ID is required", http.StatusBadRequest)
		return
	}

	// Convert recurring chore ID from string to ObjectID
	recurringChoreID, err := primitive.ObjectIDFromHex(request.RecurringChoreID)
	if err != nil {
		http.Error(w, "Invalid recurring chore ID format", http.StatusBadRequest)
		return
	}

	// Get existing recurring chore
	var recurringChore models.RecurringChore
	err = config.DB.Collection("recurring_chores").FindOne(
		context.Background(),
		bson.M{"_id": recurringChoreID},
	).Decode(&recurringChore)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Recurring chore not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch recurring chore", http.StatusInternalServerError)
		}
		return
	}

	// Validate frequency if provided
	if request.Frequency != "" {
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
	}

	// Prepare update fields
	updateFields := bson.M{
		"updated_at": time.Now(),
	}

	// Only update fields that were provided
	if request.Title != "" {
		updateFields["title"] = request.Title
	}

	if request.Description != "" {
		updateFields["description"] = request.Description
	}

	if request.Frequency != "" {
		updateFields["frequency"] = request.Frequency
	}

	if request.Points > 0 {
		updateFields["points"] = request.Points
	}

	if request.IsActive != nil {
		updateFields["is_active"] = *request.IsActive
	}

	// Update recurring chore in the database
	result, err := config.DB.Collection("recurring_chores").UpdateOne(
		context.Background(),
		bson.M{"_id": recurringChoreID},
		bson.M{"$set": updateFields},
	)

	if err != nil {
		log.Printf("Failed to update recurring chore: %v", err)
		http.Error(w, "Failed to update recurring chore", http.StatusInternalServerError)
		return
	}

	if result.ModifiedCount == 0 {
		http.Error(w, "No changes were made", http.StatusOK)
		return
	}

	// Get updated recurring chore
	var updatedRecurringChore models.RecurringChore
	err = config.DB.Collection("recurring_chores").FindOne(
		context.Background(),
		bson.M{"_id": recurringChoreID},
	).Decode(&updatedRecurringChore)

	if err != nil {
		log.Printf("Failed to fetch updated recurring chore: %v", err)
		// Return success message even though we couldn't fetch the updated document
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Recurring chore updated successfully",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedRecurringChore)
}

// DeleteRecurringChoreHandler handles deleting a recurring chore
func DeleteRecurringChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recurringChoreID := r.URL.Query().Get("recurring_chore_id")
	if recurringChoreID == "" {
		http.Error(w, "Recurring chore ID is required", http.StatusBadRequest)
		return
	}

	// Convert recurring chore ID from string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(recurringChoreID)
	if err != nil {
		http.Error(w, "Invalid recurring chore ID format", http.StatusBadRequest)
		return
	}

	// Start MongoDB session for transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Failed to start MongoDB session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Define the transaction
	_, err = session.WithTransaction(context.Background(), func(sessionContext mongo.SessionContext) (interface{}, error) {
		// Delete the recurring chore
		result, err := config.DB.Collection("recurring_chores").DeleteOne(
			sessionContext,
			bson.M{"_id": objectID},
		)

		if err != nil {
			return nil, err
		}

		if result.DeletedCount == 0 {
			return nil, errors.New("recurring chore not found")
		}

		// Delete any pending instances of this recurring chore
		_, err = config.DB.Collection("chores").DeleteMany(
			sessionContext,
			bson.M{
				"recurring_id": objectID,
				"status":       models.ChoreStatusPending,
			},
		)

		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		if err.Error() == "recurring chore not found" {
			http.Error(w, "Recurring chore not found", http.StatusNotFound)
		} else {
			log.Printf("Transaction failed: %v", err)
			http.Error(w, "Failed to delete recurring chore", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Recurring chore and pending instances deleted successfully",
	})
}
