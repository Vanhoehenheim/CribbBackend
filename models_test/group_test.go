package models_test

import (
	"cribb-backend/models"
	"testing"
)

func TestNewGroup(t *testing.T) {
	// Test data
	name := "Test Apartment"

	// Create a new group
	group := models.NewGroup(name)

	// Verify the group properties
	if group.Name != name {
		t.Errorf("Expected name %s, got %s", name, group.Name)
	}
	if len(group.GroupCode) != 6 {
		t.Errorf("Expected group code length 6, got %d", len(group.GroupCode))
	}
	if len(group.Members) != 0 {
		t.Errorf("Expected empty members array, got %d members", len(group.Members))
	}
}

func TestGenerateGroupCode(t *testing.T) {
	// Create 10 groups and verify their group codes are unique
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		group := models.NewGroup("Test " + string(rune('A'+i)))
		if codes[group.GroupCode] {
			t.Errorf("Duplicate group code generated: %s", group.GroupCode)
		}
		codes[group.GroupCode] = true

		// Verify the code format (all uppercase letters)
		for _, c := range group.GroupCode {
			if c < 'A' || c > 'Z' {
				t.Errorf("Invalid character in group code: %c", c)
			}
		}
	}
}
