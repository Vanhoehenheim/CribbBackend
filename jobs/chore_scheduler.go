// jobs/chore_scheduler.go
package jobs

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/models"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// StartChoreScheduler initializes and starts the recurring chore scheduler
func StartChoreScheduler() {
	log.Println("Starting chore scheduler...")

	// Run the scheduler every hour
	ticker := time.NewTicker(1 * time.Hour)

	// Run immediately once at startup
	go processRecurringChores()
	go detectOverdueChores()

	// Then run on the schedule
	go func() {
		for range ticker.C {
			processRecurringChores()
			detectOverdueChores()
		}
	}()
}

// processRecurringChores checks for recurring chores that need new instances created
func processRecurringChores() {
	log.Println("Processing recurring chores...")

	// Find all active recurring chores that need to create new instances
	now := time.Now()
	cursor, err := config.DB.Collection("recurring_chores").Find(
		context.Background(),
		bson.M{
			"is_active":       true,
			"next_assignment": bson.M{"$lte": now},
		},
	)

	if err != nil {
		log.Printf("Error finding recurring chores: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var recurringChores []models.RecurringChore
	if err = cursor.All(context.Background(), &recurringChores); err != nil {
		log.Printf("Error decoding recurring chores: %v", err)
		return
	}

	for _, recurringChore := range recurringChores {
		// Start a session for each recurring chore
		session, err := config.DB.Client().StartSession()
		if err != nil {
			log.Printf("Error starting session for recurring chore %s: %v", recurringChore.ID.Hex(), err)
			continue
		}

		// Use a closure to handle the session
		func(s mongo.Session, rc models.RecurringChore) {
			defer s.EndSession(context.Background())

			// Execute in a transaction
			_, err := s.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
				// Get fresh copy of recurring chore to avoid race conditions
				var freshRC models.RecurringChore
				err := config.DB.Collection("recurring_chores").FindOne(
					ctx,
					bson.M{"_id": rc.ID},
				).Decode(&freshRC)

				if err != nil {
					return nil, err
				}

				// If someone else already processed this or it's not active anymore, skip
				if freshRC.NextAssignment.After(now) || !freshRC.IsActive {
					return nil, nil
				}

				// Create a new chore instance
				newChore := models.CreateChoreFromRecurring(&freshRC)
				_, err = config.DB.Collection("chores").InsertOne(ctx, newChore)
				if err != nil {
					return nil, err
				}

				// Calculate next assignment date
				var nextAssignment time.Time
				switch freshRC.Frequency {
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

				// Update the recurring chore with the new next assignment date
				_, err = config.DB.Collection("recurring_chores").UpdateOne(
					ctx,
					bson.M{"_id": freshRC.ID},
					bson.M{
						"$set": bson.M{
							"next_assignment": nextAssignment,
							"updated_at":      time.Now(),
						},
					},
				)

				if err != nil {
					return nil, err
				}

				log.Printf("Created new chore instance from recurring chore %s", freshRC.ID.Hex())
				return nil, nil
			})

			if err != nil {
				log.Printf("Error processing recurring chore %s: %v", rc.ID.Hex(), err)
			}
		}(session, recurringChore)
	}

	log.Printf("Processed %d recurring chores", len(recurringChores))
}

// detectOverdueChores finds and marks overdue chores
func detectOverdueChores() {
	log.Println("Detecting overdue chores...")

	// Calculate the start of today in UTC
	year, month, day := time.Now().UTC().Date()
	startOfTodayUTC := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	// Find all pending chores where the start of today UTC is
	// at or after the start of the day FOLLOWING the chore's due date.
	// This means the entire due date has passed.
	// We can't directly compare with AddDate in BSON, so we query
	// for due dates less than the start of *yesterday*.
	startOfYesterdayUTC := startOfTodayUTC.AddDate(0, 0, -1)

	result, err := config.DB.Collection("chores").UpdateMany(
		context.Background(),
		bson.M{
			"status": models.ChoreStatusPending,
			// Due date is strictly less than the start of yesterday UTC
			"due_date": bson.M{"$lt": startOfYesterdayUTC},
		},
		bson.M{
			"$set": bson.M{
				"status":     models.ChoreStatusOverdue,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		log.Printf("Error updating overdue chores: %v", err)
		return
	}

	if result.ModifiedCount > 0 {
		log.Printf("Marked %d chores as overdue", result.ModifiedCount)
	} else {
		log.Printf("No overdue chores found")
	}
}
