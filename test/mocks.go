// test/mocks.go
package test

import (
	"cribb-backend/models"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDB provides a simplified interface for testing database operations
type TestDB struct {
	Users               []models.User
	Groups              []models.Group
	Chores              []models.Chore
	RecurringChores     []models.RecurringChore
	ChoreCompletions    []models.ChoreCompletion
	PantryItems         []models.PantryItem         // Added for pantry tests
	PantryNotifications []models.PantryNotification // Added for pantry tests
	PantryHistory       []models.PantryHistory      // Added for pantry tests
	ShoppingCartItems   []models.ShoppingCartItem
}

// NewTestDB creates a new test database with some initial data
func NewTestDB() *TestDB {
	return &TestDB{
		Users:               []models.User{},
		Groups:              []models.Group{},
		Chores:              []models.Chore{},
		RecurringChores:     []models.RecurringChore{},
		ChoreCompletions:    []models.ChoreCompletion{},
		PantryItems:         []models.PantryItem{},
		PantryNotifications: []models.PantryNotification{},
		PantryHistory:       []models.PantryHistory{},
		ShoppingCartItems:   []models.ShoppingCartItem{},
	}
}

// AddUser adds a user to the test database
func (db *TestDB) AddUser(user models.User) {
	db.Users = append(db.Users, user)
}

// AddGroup adds a group to the test database
func (db *TestDB) AddGroup(group models.Group) {
	db.Groups = append(db.Groups, group)
}

// AddChore adds a chore to the test database
func (db *TestDB) AddChore(chore models.Chore) {
	db.Chores = append(db.Chores, chore)
}

// AddRecurringChore adds a recurring chore to the test database
func (db *TestDB) AddRecurringChore(chore models.RecurringChore) {
	db.RecurringChores = append(db.RecurringChores, chore)
}

// AddChoreCompletion adds a chore completion to the test database
func (db *TestDB) AddChoreCompletion(completion models.ChoreCompletion) {
	db.ChoreCompletions = append(db.ChoreCompletions, completion)
}

// FindUserByUsername finds a user by username
func (db *TestDB) FindUserByUsername(username string) (models.User, bool) {
	for _, user := range db.Users {
		if user.Username == username {
			return user, true
		}
	}
	return models.User{}, false
}

// FindUserByID finds a user by ID
func (db *TestDB) FindUserByID(id primitive.ObjectID) (models.User, bool) {
	for _, user := range db.Users {
		if user.ID == id {
			return user, true
		}
	}
	return models.User{}, false
}

// FindGroupByName finds a group by name
func (db *TestDB) FindGroupByName(name string) (models.Group, bool) {
	for _, group := range db.Groups {
		if group.Name == name {
			return group, true
		}
	}
	return models.Group{}, false
}

// FindGroupByCode finds a group by code
func (db *TestDB) FindGroupByCode(code string) (models.Group, bool) {
	for _, group := range db.Groups {
		if group.GroupCode == code {
			return group, true
		}
	}
	return models.Group{}, false
}

// UpdateGroup updates a group in the test database
func (db *TestDB) UpdateGroup(group models.Group) {
	for i, g := range db.Groups {
		if g.ID == group.ID {
			db.Groups[i] = group
			return
		}
	}
	// If not found, add it
	db.AddGroup(group)
}

// FindChoreByID finds a chore by ID
func (db *TestDB) FindChoreByID(id primitive.ObjectID) (models.Chore, bool) {
	for _, chore := range db.Chores {
		if chore.ID == id {
			return chore, true
		}
	}
	return models.Chore{}, false
}

// FindRecurringChoreByID finds a recurring chore by ID
func (db *TestDB) FindRecurringChoreByID(id primitive.ObjectID) (models.RecurringChore, bool) {
	for _, chore := range db.RecurringChores {
		if chore.ID == id {
			return chore, true
		}
	}
	return models.RecurringChore{}, false
}

// GetChoresForUser gets all chores assigned to a user
func (db *TestDB) GetChoresForUser(userID primitive.ObjectID) []models.Chore {
	var result []models.Chore
	for _, chore := range db.Chores {
		if chore.AssignedTo == userID {
			result = append(result, chore)
		}
	}
	return result
}

// GetChoresForGroup gets all chores for a group
func (db *TestDB) GetChoresForGroup(groupID primitive.ObjectID) []models.Chore {
	var result []models.Chore
	for _, chore := range db.Chores {
		if chore.GroupID == groupID {
			result = append(result, chore)
		}
	}
	return result
}

// GetRecurringChoresForGroup gets all recurring chores for a group
func (db *TestDB) GetRecurringChoresForGroup(groupID primitive.ObjectID) []models.RecurringChore {
	var result []models.RecurringChore
	for _, chore := range db.RecurringChores {
		if chore.GroupID == groupID {
			result = append(result, chore)
		}
	}
	return result
}

// GetUsersForGroup gets all users in a group
func (db *TestDB) GetUsersForGroup(groupID primitive.ObjectID) []models.User {
	var result []models.User
	for _, user := range db.Users {
		if user.GroupID == groupID {
			result = append(result, user)
		}
	}
	return result
}

// UpdateUser updates a user in the test database
func (db *TestDB) UpdateUser(user models.User) {
	for i, u := range db.Users {
		if u.ID == user.ID {
			db.Users[i] = user
			return
		}
	}
	// If not found, add it
	db.AddUser(user)
}

// UpdateChore updates a chore in the test database
func (db *TestDB) UpdateChore(chore models.Chore) {
	for i, c := range db.Chores {
		if c.ID == chore.ID {
			db.Chores[i] = chore
			return
		}
	}
	// If not found, add it
	db.AddChore(chore)
}

// UpdateRecurringChore updates a recurring chore in the test database
func (db *TestDB) UpdateRecurringChore(chore models.RecurringChore) {
	for i, c := range db.RecurringChores {
		if c.ID == chore.ID {
			db.RecurringChores[i] = chore
			return
		}
	}
	// If not found, add it
	db.AddRecurringChore(chore)
}

// DeleteChore deletes a chore from the test database
func (db *TestDB) DeleteChore(id primitive.ObjectID) bool {
	for i, c := range db.Chores {
		if c.ID == id {
			db.Chores = append(db.Chores[:i], db.Chores[i+1:]...)
			return true
		}
	}
	return false
}

// DeleteRecurringChore deletes a recurring chore from the test database
func (db *TestDB) DeleteRecurringChore(id primitive.ObjectID) bool {
	for i, c := range db.RecurringChores {
		if c.ID == id {
			db.RecurringChores = append(db.RecurringChores[:i], db.RecurringChores[i+1:]...)
			return true
		}
	}
	return false
}

// Helper functions for creating test data
func CreateTestUser() models.User {
	return models.User{
		ID:          primitive.NewObjectID(),
		Username:    "testuser",
		Password:    "$2a$10$abcdefghijklmnopqrstuvwxyz012345",
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
}

func CreateTestGroup() models.Group {
	return models.Group{
		ID:        primitive.NewObjectID(),
		Name:      "Test Apartment",
		GroupCode: "ABCDEF",
		Members:   []primitive.ObjectID{primitive.NewObjectID()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func CreateTestChore() models.Chore {
	return models.Chore{
		ID:          primitive.NewObjectID(),
		Title:       "Clean Kitchen",
		Description: "Wash dishes and wipe counters",
		Type:        models.ChoreTypeIndividual,
		GroupID:     primitive.NewObjectID(),
		AssignedTo:  primitive.NewObjectID(),
		Status:      models.ChoreStatusPending,
		Points:      10,
		StartDate:   time.Now(),
		DueDate:     time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func CreateTestRecurringChore() models.RecurringChore {
	return models.RecurringChore{
		ID:             primitive.NewObjectID(),
		Title:          "Take Out Trash",
		Description:    "Empty trash bins",
		GroupID:        primitive.NewObjectID(),
		MemberRotation: []primitive.ObjectID{primitive.NewObjectID(), primitive.NewObjectID()},
		CurrentIndex:   0,
		Frequency:      "weekly",
		Points:         5,
		NextAssignment: time.Now().Add(7 * 24 * time.Hour),
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}
