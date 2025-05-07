// handlers/chore_completion.go
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
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CompleteChoreHandler handles the completion of a chore by a user
func CompleteChoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ChoreID string `json:"chore_id"`
		UserID  string `json:"user_id"` // Changed from Username to UserID
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.ChoreID == "" || request.UserID == "" {
		http.Error(w, "Chore ID and user ID are required", http.StatusBadRequest)
		return
	}

	// Convert chore ID from string to ObjectID
	choreID, err := primitive.ObjectIDFromHex(request.ChoreID)
	if err != nil {
		http.Error(w, "Invalid chore ID format", http.StatusBadRequest)
		return
	}

	// Convert user ID from string to ObjectID
	userID, err := primitive.ObjectIDFromHex(request.UserID)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Start a MongoDB session for transaction
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Failed to start MongoDB session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Define the transaction
	result, err := session.WithTransaction(context.Background(), func(sessionContext mongo.SessionContext) (interface{}, error) {
		// 1. Get the user by ID
		var user models.User
		err := config.DB.Collection("users").FindOne(
			sessionContext,
			bson.M{"_id": userID},
		).Decode(&user)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, errors.New("user not found")
			}
			return nil, err
		}

		// 2. Get the chore
		var chore models.Chore
		err = config.DB.Collection("chores").FindOne(
			sessionContext,
			bson.M{"_id": choreID},
		).Decode(&chore)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, errors.New("chore not found")
			}
			return nil, err
		}

		// 3. Verify the chore is assigned to the user
		if chore.AssignedTo != userID {
			return nil, errors.New("chore is not assigned to this user")
		}

		// 4. Verify the chore is not already completed
		if chore.Status == models.ChoreStatusCompleted {
			return nil, errors.New("chore is already completed")
		}

		now := time.Now()

		// 5. Create chore completion record
		choreCompletion := models.ChoreCompletion{
			ChoreID:     chore.ID,
			UserID:      user.ID,
			CompletedAt: now,
			Points:      chore.Points,
		}

		_, err = config.DB.Collection("chore_completions").InsertOne(
			sessionContext,
			choreCompletion,
		)
		if err != nil {
			return nil, err
		}

		// 6. Update chore status to completed
		_, err = config.DB.Collection("chores").UpdateOne(
			sessionContext,
			bson.M{"_id": chore.ID},
			bson.M{
				"$set": bson.M{
					"status":     models.ChoreStatusCompleted,
					"updated_at": now,
				},
			},
		)
		if err != nil {
			return nil, err
		}

		// 7. Update user's score
		_, err = config.DB.Collection("users").UpdateOne(
			sessionContext,
			bson.M{"_id": user.ID},
			bson.M{
				"$inc": bson.M{"score": chore.Points},
				"$set": bson.M{"updated_at": now},
			},
		)
		if err != nil {
			return nil, err
		}

		// 8. If this is a recurring chore, create the next instance
		if chore.Type == models.ChoreTypeRecurring && !chore.RecurringID.IsZero() {
			var recurringChore models.RecurringChore
			err = config.DB.Collection("recurring_chores").FindOne(
				sessionContext,
				bson.M{"_id": chore.RecurringID},
			).Decode(&recurringChore)

			if err == nil && recurringChore.IsActive {
				// Calculate the next assignment date
				var nextAssignment time.Time
				switch recurringChore.Frequency {
				case "daily":
					nextAssignment = now.Add(24 * time.Hour)
				case "weekly":
					nextAssignment = now.Add(7 * 24 * time.Hour)
				case "biweekly":
					nextAssignment = now.Add(14 * 24 * time.Hour)
				case "monthly":
					nextAssignment = now.AddDate(0, 1, 0)
				default:
					nextAssignment = now.Add(7 * 24 * time.Hour) // Default to weekly
				}

				// Update recurring chore with next assignment date
				_, err = config.DB.Collection("recurring_chores").UpdateOne(
					sessionContext,
					bson.M{"_id": recurringChore.ID},
					bson.M{
						"$set": bson.M{
							"next_assignment": nextAssignment,
							"updated_at":      now,
						},
					},
				)
				if err != nil {
					return nil, err
				}

				// Create next chore instance
				nextChore := models.CreateChoreFromRecurring(&recurringChore)
				_, err = config.DB.Collection("chores").InsertOne(sessionContext, nextChore)
				if err != nil {
					return nil, err
				}
			}
		}

		return map[string]interface{}{
			"points_earned": chore.Points,
			"new_score":     user.Score + chore.Points,
		}, nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetGroupChoresHandler retrieves all active chores for a group
// GetGroupChoresHandler retrieves all active chores for a group
func GetGroupChoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupName := r.URL.Query().Get("group_name")
	if groupName == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Find the group by name
	var group models.Group
	err := config.DB.Collection("groups").FindOne(
		context.Background(),
		bson.M{"name": groupName},
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Get all chores for the group, sorted by due date
	opts := options.Find().SetSort(bson.D{{Key: "due_date", Value: 1}})
	cursor, err := config.DB.Collection("chores").Find(
		context.Background(),
		bson.M{"group_id": group.ID},
		opts,
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
	// now := time.Now()
	// Calculate the start of today in UTC
	year, month, day := time.Now().UTC().Date()
	startOfTodayUTC := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	for i, chore := range chores {
		// Remove previous logging
		// log.Printf(...)

		if chore.Status != models.ChoreStatusOverdue &&
			chore.Status != models.ChoreStatusCompleted &&
			!chore.DueDate.IsZero() {

			// Calculate the start of the day AFTER the due date
			year, month, day := chore.DueDate.Date()
			startOfDueDayUTC := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			endOfDueDateUTC := startOfDueDayUTC.AddDate(0, 0, 1) // Start of the next day

			// Check if the current time is at or after the start of the day following the due date
			if !startOfTodayUTC.Before(endOfDueDateUTC) { // Equivalent to startOfTodayUTC >= endOfDueDateUTC
				// log.Printf("Marking chore ID %s as OVERDUE", chore.ID.Hex()) // Optional: keep logging if needed
				chores[i].Status = models.ChoreStatusOverdue

				// Update in database (don't wait for the result)
				go func(choreID primitive.ObjectID) {
					_, err := config.DB.Collection("chores").UpdateOne(
						context.Background(),
						bson.M{"_id": choreID},
						bson.M{"$set": bson.M{"status": models.ChoreStatusOverdue}},
					)
					if err != nil {
						log.Printf("Failed to update chore status to overdue: %v", err)
					}
				}(chore.ID)
			}
		}
	}

	// For each chore, include assignee information
	type ChoreWithAssignee struct {
		models.Chore
		AssigneeName string `json:"assignee_name"`
	}

	choresWithAssignees := make([]ChoreWithAssignee, 0, len(chores))

	for _, chore := range chores {
		choreWithAssignee := ChoreWithAssignee{
			Chore:        chore,
			AssigneeName: "",
		}

		if !chore.AssignedTo.IsZero() {
			var user models.User
			err := config.DB.Collection("users").FindOne(
				context.Background(),
				bson.M{"_id": chore.AssignedTo},
			).Decode(&user)

			if err == nil {
				choreWithAssignee.AssigneeName = user.Name
			}
		}

		choresWithAssignees = append(choresWithAssignees, choreWithAssignee)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(choresWithAssignees)
}

// GetGroupRecurringChoresHandler retrieves all recurring chores for a group
func GetGroupRecurringChoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupName := r.URL.Query().Get("group_name")
	if groupName == "" {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	// Find the group by name
	var group models.Group
	err := config.DB.Collection("groups").FindOne(
		context.Background(),
		bson.M{"name": groupName},
	).Decode(&group)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Get all recurring chores for the group
	cursor, err := config.DB.Collection("recurring_chores").Find(
		context.Background(),
		bson.M{
			"group_id":  group.ID,
			"is_active": true,
		},
	)
	if err != nil {
		http.Error(w, "Failed to fetch recurring chores", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var recurringChores []models.RecurringChore
	if err = cursor.All(context.Background(), &recurringChores); err != nil {
		http.Error(w, "Failed to decode recurring chores", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recurringChores)
}
