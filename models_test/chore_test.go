package models_test

import (
	"cribb-backend/models"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateChore(t *testing.T) {
	// Test data
	title := "Test Chore"
	description := "Test Description"
	groupID := primitive.NewObjectID()
	assignedTo := primitive.NewObjectID()
	dueDate := time.Now().Add(24 * time.Hour)
	points := 10

	// Create a chore
	chore := models.CreateChore(title, description, groupID, assignedTo, dueDate, points)

	// Verify the chore properties
	if chore.Title != title {
		t.Errorf("Expected title %s, got %s", title, chore.Title)
	}
	if chore.Description != description {
		t.Errorf("Expected description %s, got %s", description, chore.Description)
	}
	if chore.GroupID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID.Hex(), chore.GroupID.Hex())
	}
	if chore.AssignedTo != assignedTo {
		t.Errorf("Expected assigned to %s, got %s", assignedTo.Hex(), chore.AssignedTo.Hex())
	}
	if chore.Points != points {
		t.Errorf("Expected points %d, got %d", points, chore.Points)
	}
	if chore.Status != models.ChoreStatusPending {
		t.Errorf("Expected status %s, got %s", models.ChoreStatusPending, chore.Status)
	}
	if chore.Type != models.ChoreTypeIndividual {
		t.Errorf("Expected type %s, got %s", models.ChoreTypeIndividual, chore.Type)
	}
	if !chore.DueDate.Equal(dueDate) {
		t.Errorf("Expected due date %v, got %v", dueDate, chore.DueDate)
	}
}

func TestCreateRecurringChore(t *testing.T) {
	// Test data
	title := "Test Recurring Chore"
	description := "Test Description"
	groupID := primitive.NewObjectID()
	memberRotation := []primitive.ObjectID{primitive.NewObjectID(), primitive.NewObjectID()}
	frequency := "weekly"
	points := 5

	// Create a recurring chore
	recurringChore := models.CreateRecurringChore(title, description, groupID, memberRotation, frequency, points)

	// Verify the recurring chore properties
	if recurringChore.Title != title {
		t.Errorf("Expected title %s, got %s", title, recurringChore.Title)
	}
	if recurringChore.Description != description {
		t.Errorf("Expected description %s, got %s", description, recurringChore.Description)
	}
	if recurringChore.GroupID != groupID {
		t.Errorf("Expected group ID %s, got %s", groupID.Hex(), recurringChore.GroupID.Hex())
	}
	if len(recurringChore.MemberRotation) != len(memberRotation) {
		t.Errorf("Expected member rotation length %d, got %d", len(memberRotation), len(recurringChore.MemberRotation))
	}
	if recurringChore.Frequency != frequency {
		t.Errorf("Expected frequency %s, got %s", frequency, recurringChore.Frequency)
	}
	if recurringChore.Points != points {
		t.Errorf("Expected points %d, got %d", points, recurringChore.Points)
	}
	if recurringChore.CurrentIndex != 0 {
		t.Errorf("Expected current index 0, got %d", recurringChore.CurrentIndex)
	}
	if !recurringChore.IsActive {
		t.Errorf("Expected is_active to be true")
	}
}

func TestGetNextAssignee(t *testing.T) {
	// Create member rotation
	member1 := primitive.NewObjectID()
	member2 := primitive.NewObjectID()
	member3 := primitive.NewObjectID()
	memberRotation := []primitive.ObjectID{member1, member2, member3}

	// Create recurring chore
	recurringChore := models.CreateRecurringChore(
		"Test Chore",
		"Test Description",
		primitive.NewObjectID(),
		memberRotation,
		"weekly",
		5,
	)

	// Initial index should be 0
	if recurringChore.CurrentIndex != 0 {
		t.Errorf("Expected initial index 0, got %d", recurringChore.CurrentIndex)
	}

	// First call should return member1 and advance index to 1
	assignee := recurringChore.GetNextAssignee()
	if assignee != member1 {
		t.Errorf("Expected first assignee to be %s, got %s", member1.Hex(), assignee.Hex())
	}
	if recurringChore.CurrentIndex != 1 {
		t.Errorf("Expected index to advance to 1, got %d", recurringChore.CurrentIndex)
	}

	// Second call should return member2 and advance index to 2
	assignee = recurringChore.GetNextAssignee()
	if assignee != member2 {
		t.Errorf("Expected second assignee to be %s, got %s", member2.Hex(), assignee.Hex())
	}
	if recurringChore.CurrentIndex != 2 {
		t.Errorf("Expected index to advance to 2, got %d", recurringChore.CurrentIndex)
	}

	// Third call should return member3 and advance index to 0 (wrap around)
	assignee = recurringChore.GetNextAssignee()
	if assignee != member3 {
		t.Errorf("Expected third assignee to be %s, got %s", member3.Hex(), assignee.Hex())
	}
	if recurringChore.CurrentIndex != 0 {
		t.Errorf("Expected index to wrap around to 0, got %d", recurringChore.CurrentIndex)
	}

	// Fourth call should return member1 again
	assignee = recurringChore.GetNextAssignee()
	if assignee != member1 {
		t.Errorf("Expected fourth assignee to be %s, got %s", member1.Hex(), assignee.Hex())
	}
}

func TestCreateChoreFromRecurring(t *testing.T) {
	// Create recurring chore
	recurringChore := models.RecurringChore{
		ID:             primitive.NewObjectID(),
		Title:          "Test Recurring Chore",
		Description:    "Test Description",
		GroupID:        primitive.NewObjectID(),
		MemberRotation: []primitive.ObjectID{primitive.NewObjectID(), primitive.NewObjectID()},
		CurrentIndex:   0,
		Frequency:      "weekly",
		Points:         5,
		NextAssignment: time.Now(),
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create chore from recurring
	chore := models.CreateChoreFromRecurring(&recurringChore)

	// Verify the chore properties
	if chore.Title != recurringChore.Title {
		t.Errorf("Expected title %s, got %s", recurringChore.Title, chore.Title)
	}
	if chore.Description != recurringChore.Description {
		t.Errorf("Expected description %s, got %s", recurringChore.Description, chore.Description)
	}
	if chore.GroupID != recurringChore.GroupID {
		t.Errorf("Expected group ID %s, got %s", recurringChore.GroupID.Hex(), chore.GroupID.Hex())
	}
	if chore.Points != recurringChore.Points {
		t.Errorf("Expected points %d, got %d", recurringChore.Points, chore.Points)
	}
	if chore.Status != models.ChoreStatusPending {
		t.Errorf("Expected status %s, got %s", models.ChoreStatusPending, chore.Status)
	}
	if chore.Type != models.ChoreTypeRecurring {
		t.Errorf("Expected type %s, got %s", models.ChoreTypeRecurring, chore.Type)
	}
	if chore.RecurringID != recurringChore.ID {
		t.Errorf("Expected recurring ID %s, got %s", recurringChore.ID.Hex(), chore.RecurringID.Hex())
	}

	// Due date should be calculated based on frequency
	now := time.Now()
	var expectedDueDate time.Time
	switch recurringChore.Frequency {
	case "daily":
		expectedDueDate = now.Add(24 * time.Hour)
	case "weekly":
		expectedDueDate = now.Add(7 * 24 * time.Hour)
	case "biweekly":
		expectedDueDate = now.Add(14 * 24 * time.Hour)
	case "monthly":
		expectedDueDate = now.AddDate(0, 1, 0)
	}

	// Allow a small time difference due to execution time
	timeDiff := chore.DueDate.Sub(expectedDueDate)
	if timeDiff < -5*time.Second || timeDiff > 5*time.Second {
		t.Errorf("Expected due date around %v, got %v (diff: %v)", expectedDueDate, chore.DueDate, timeDiff)
	}
}
