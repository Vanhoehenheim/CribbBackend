package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChoreType represents the type of chore
type ChoreType string

const (
	ChoreTypeRecurring  ChoreType = "recurring"  // Shared chore that rotates among group members
	ChoreTypeIndividual ChoreType = "individual" // Individual chore assigned to specific user
)

// ChoreStatus represents the status of a chore
type ChoreStatus string

const (
	ChoreStatusPending   ChoreStatus = "pending"
	ChoreStatusCompleted ChoreStatus = "completed"
	ChoreStatusOverdue   ChoreStatus = "overdue"
)

// Chore represents a task that needs to be completed
type Chore struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title" validate:"required"`
	Description string             `bson:"description" json:"description"`
	Type        ChoreType          `bson:"type" json:"type" validate:"required"`
	GroupID     primitive.ObjectID `bson:"group_id" json:"group_id" validate:"required"`
	AssignedTo  primitive.ObjectID `bson:"assigned_to,omitempty" json:"assigned_to,omitempty"`
	Status      ChoreStatus        `bson:"status" json:"status"`
	Points      int                `bson:"points" json:"points" validate:"required,min=1"`
	StartDate   time.Time          `bson:"start_date" json:"start_date"`
	DueDate     time.Time          `bson:"due_date,omitempty" json:"due_date,omitempty"`
	RecurringID primitive.ObjectID `bson:"recurring_id,omitempty" json:"recurring_id,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// RecurringChore represents a template for chores that rotate among group members
type RecurringChore struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Title          string               `bson:"title" json:"title" validate:"required"`
	Description    string               `bson:"description" json:"description"`
	GroupID        primitive.ObjectID   `bson:"group_id" json:"group_id" validate:"required"`
	MemberRotation []primitive.ObjectID `bson:"member_rotation" json:"member_rotation"` // Order of members for rotation
	CurrentIndex   int                  `bson:"current_index" json:"current_index"`     // Current position in rotation
	Frequency      string               `bson:"frequency" json:"frequency"`             // daily, weekly, etc.
	Points         int                  `bson:"points" json:"points" validate:"required,min=1"`
	NextAssignment time.Time            `bson:"next_assignment" json:"next_assignment"` // When the next chore should be assigned
	IsActive       bool                 `bson:"is_active" json:"is_active"`
	CreatedAt      time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time            `bson:"updated_at" json:"updated_at"`
}

// ChoreCompletion represents a record of a completed chore
type ChoreCompletion struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChoreID     primitive.ObjectID `bson:"chore_id" json:"chore_id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	CompletedAt time.Time          `bson:"completed_at" json:"completed_at"`
	Points      int                `bson:"points" json:"points"`
}

// CreateChore creates a new individual chore
func CreateChore(title, description string, groupID, assignedTo primitive.ObjectID, dueDate time.Time, points int) *Chore {
	return &Chore{
		Title:       title,
		Description: description,
		Type:        ChoreTypeIndividual,
		GroupID:     groupID,
		AssignedTo:  assignedTo,
		Status:      ChoreStatusPending,
		Points:      points,
		StartDate:   time.Now(),
		DueDate:     dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateRecurringChore creates a new recurring chore definition
func CreateRecurringChore(title, description string, groupID primitive.ObjectID, memberRotation []primitive.ObjectID, frequency string, points int) *RecurringChore {
	return &RecurringChore{
		Title:          title,
		Description:    description,
		GroupID:        groupID,
		MemberRotation: memberRotation,
		CurrentIndex:   0,
		Frequency:      frequency,
		Points:         points,
		NextAssignment: time.Now(),
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// GetNextAssignee returns the next user ID in the rotation
func (rc *RecurringChore) GetNextAssignee() primitive.ObjectID {
	if len(rc.MemberRotation) == 0 {
		return primitive.NilObjectID
	}

	assignee := rc.MemberRotation[rc.CurrentIndex]
	rc.CurrentIndex = (rc.CurrentIndex + 1) % len(rc.MemberRotation)
	return assignee
}

// CreateChoreFromRecurring creates a new chore instance from a recurring chore
func CreateChoreFromRecurring(recurringChore *RecurringChore) *Chore {
	// Get the next assignee
	assignedTo := recurringChore.GetNextAssignee()

	// Calculate due date based on frequency
	dueDate := time.Now()
	switch recurringChore.Frequency {
	case "daily":
		dueDate = dueDate.Add(24 * time.Hour)
	case "weekly":
		dueDate = dueDate.Add(7 * 24 * time.Hour)
	case "biweekly":
		dueDate = dueDate.Add(14 * 24 * time.Hour)
	case "monthly":
		dueDate = dueDate.AddDate(0, 1, 0)
	}

	return &Chore{
		Title:       recurringChore.Title,
		Description: recurringChore.Description,
		Type:        ChoreTypeRecurring,
		GroupID:     recurringChore.GroupID,
		AssignedTo:  assignedTo,
		Status:      ChoreStatusPending,
		Points:      recurringChore.Points,
		StartDate:   time.Now(),
		DueDate:     dueDate,
		RecurringID: recurringChore.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
