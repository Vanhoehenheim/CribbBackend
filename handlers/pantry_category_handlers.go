package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/middleware"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateCategoryRequest defines the request structure for creating a custom category
type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

// UpdateCategoryRequest defines the request structure for updating a custom category
type UpdateCategoryRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

// CategoryWithCreator represents a category with creator information
type CategoryWithCreator struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	GroupID     *string            `json:"group_id,omitempty"`
	GroupName   *string            `json:"group_name,omitempty"`
	CreatedBy   *string            `json:"created_by,omitempty"`
	CreatedByID *string            `json:"created_by_id,omitempty"`
}

// StructuredCategoryResponse represents the new structured response format
type StructuredCategoryResponse struct {
	GroupName  string           `json:"group_name"`
	GroupID    string           `json:"group_id"`
	Categories CategoriesByType `json:"categories"`
}

type CategoriesByType struct {
	Predefined  []CategoryWithCreator `json:"predefined"`
	UserDefined []CategoryWithCreator `json:"user_defined"`
}

// CategoryResponse defines the response structure for category operations
type CategoryResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// GetPantryCategoriesHandler retrieves all available categories for a group in structured format
func GetPantryCategoriesHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get group_name from query parameter
	groupName := r.URL.Query().Get("group_name")
	if groupName == "" {
		http.Error(w, "group_name parameter is required", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the group by name
	var group models.Group
	err = config.DB.Collection("groups").FindOne(
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

	// Verify user belongs to the group
	if user.GroupID != group.ID {
		http.Error(w, "User is not a member of this group", http.StatusForbidden)
		return
	}

	// Query for categories: predefined OR custom categories for user's group
	filter := bson.M{
		"is_active": true,
		"$or": []bson.M{
			{"type": models.CategoryTypePredefined},
			{
				"type":     models.CategoryTypeCustom,
				"group_id": user.GroupID,
			},
		},
	}

	// Sort by type (predefined first) then by name
	opts := options.Find().SetSort(bson.D{
		{Key: "type", Value: 1}, // predefined comes before custom alphabetically
		{Key: "name", Value: 1},
	})

	cursor, err := config.DB.Collection("pantry_categories").Find(
		context.Background(),
		filter,
		opts,
	)

	if err != nil {
		log.Printf("Failed to fetch pantry categories: %v", err)
		http.Error(w, "Failed to fetch categories", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var categories []models.PantryCategory
	if err = cursor.All(context.Background(), &categories); err != nil {
		log.Printf("Failed to decode pantry categories: %v", err)
		http.Error(w, "Failed to decode categories", http.StatusInternalServerError)
		return
	}

	// Separate categories by type and add creator information
	predefinedCategories := []CategoryWithCreator{}
	userDefinedCategories := []CategoryWithCreator{}

	for _, category := range categories {
		categoryWithCreator := CategoryWithCreator{
			ID:   category.ID,
			Name: category.Name,
			Type: string(category.Type),
		}

		if category.IsPredefined() {
			predefinedCategories = append(predefinedCategories, categoryWithCreator)
		} else {
			// For custom categories, add group and creator information
			groupIDStr := category.GroupID.Hex()
			categoryWithCreator.GroupID = &groupIDStr
			categoryWithCreator.GroupName = &group.Name

			// Get creator information
			if category.CreatedBy != nil {
				var creator models.User
				err := config.DB.Collection("users").FindOne(
					context.Background(),
					bson.M{"_id": *category.CreatedBy},
				).Decode(&creator)

				if err == nil {
					createdByIDStr := creator.ID.Hex()
					categoryWithCreator.CreatedBy = &creator.Name
					categoryWithCreator.CreatedByID = &createdByIDStr
				}
			}

			userDefinedCategories = append(userDefinedCategories, categoryWithCreator)
		}
	}

	// Build structured response
	response := StructuredCategoryResponse{
		GroupName: group.Name,
		GroupID:   group.ID.Hex(),
		Categories: CategoriesByType{
			Predefined:  predefinedCategories,
			UserDefined: userDefinedCategories,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreatePantryCategoryHandler creates a new custom category for the group
func CreatePantryCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	var request CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(request.Name) == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the category name
	categoryName := strings.TrimSpace(request.Name)

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Check for existing category with proper case-insensitive matching
	escapedName := regexp.QuoteMeta(categoryName)
	existingFilter := bson.M{
		"name":      bson.M{"$regex": primitive.Regex{Pattern: "^" + escapedName + "$", Options: "i"}},
		"is_active": true,
		"$or": []bson.M{
			{"type": models.CategoryTypePredefined},
			{
				"type":     models.CategoryTypeCustom,
				"group_id": user.GroupID,
			},
		},
	}

	var existingCategory models.PantryCategory
	err = config.DB.Collection("pantry_categories").FindOne(
		context.Background(),
		existingFilter,
	).Decode(&existingCategory)

	if err == nil {
		// Category already exists
		categoryType := "predefined"
		if existingCategory.IsCustom() {
			categoryType = "custom"
		}
		http.Error(w, "A "+categoryType+" category with this name already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		// Some other error occurred
		log.Printf("Error checking existing category: %v", err)
		http.Error(w, "Failed to check existing categories", http.StatusInternalServerError)
		return
	}

	// Create new custom category
	newCategory := models.CreateCustomCategory(categoryName, user.GroupID, userID)

	// Insert the category
	result, err := config.DB.Collection("pantry_categories").InsertOne(
		context.Background(),
		newCategory,
	)

	if err != nil {
		log.Printf("Failed to create custom category: %v", err)
		if mongo.IsDuplicateKeyError(err) {
			http.Error(w, "Category with this name already exists for your group", http.StatusConflict)
		} else {
			http.Error(w, "Failed to create category", http.StatusInternalServerError)
		}
		return
	}

	// Set the inserted ID
	newCategory.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CategoryResponse{
		Status:  "success",
		Message: "Custom category created successfully",
		Data:    newCategory,
	})
}

// UpdatePantryCategoryHandler updates a custom category (only group members can edit their custom ones)
func UpdatePantryCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get category ID from URL path
	categoryIDStr := strings.TrimPrefix(r.URL.Path, "/api/pantry/categories/")
	if categoryIDStr == "" {
		http.Error(w, "Category ID is required", http.StatusBadRequest)
		return
	}

	// Convert category ID to ObjectID
	categoryID, err := primitive.ObjectIDFromHex(categoryIDStr)
	if err != nil {
		http.Error(w, "Invalid category ID format", http.StatusBadRequest)
		return
	}

	var request UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(request.Name) == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	// Sanitize the category name
	categoryName := strings.TrimSpace(request.Name)

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the category to update
	var category models.PantryCategory
	err = config.DB.Collection("pantry_categories").FindOne(
		context.Background(),
		bson.M{"_id": categoryID},
	).Decode(&category)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Category not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch category", http.StatusInternalServerError)
		}
		return
	}

	// Check permissions FIRST before doing anything else
	if !category.CanBeEditedBy(userID, user.GroupID) {
		if category.IsPredefined() {
			http.Error(w, "Predefined categories cannot be edited", http.StatusForbidden)
		} else {
			http.Error(w, "You can only edit custom categories from your group", http.StatusForbidden)
		}
		return
	}

	// Check if a category with the new name already exists (excluding current category)
	escapedName := regexp.QuoteMeta(categoryName)
	existingFilter := bson.M{
		"_id":       bson.M{"$ne": categoryID},
		"name":      bson.M{"$regex": primitive.Regex{Pattern: "^" + escapedName + "$", Options: "i"}},
		"is_active": true,
		"$or": []bson.M{
			{"type": models.CategoryTypePredefined},
			{
				"type":     models.CategoryTypeCustom,
				"group_id": user.GroupID,
			},
		},
	}

	var existingCategory models.PantryCategory
	err = config.DB.Collection("pantry_categories").FindOne(
		context.Background(),
		existingFilter,
	).Decode(&existingCategory)

	if err == nil {
		// Category with new name already exists
		categoryType := "predefined"
		if existingCategory.IsCustom() {
			categoryType = "custom"
		}
		http.Error(w, "A "+categoryType+" category with this name already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		// Some other error occurred
		log.Printf("Error checking existing category: %v", err)
		http.Error(w, "Failed to check existing categories", http.StatusInternalServerError)
		return
	}

	// Update the category
	_, err = config.DB.Collection("pantry_categories").UpdateOne(
		context.Background(),
		bson.M{"_id": categoryID},
		bson.M{"$set": bson.M{"name": categoryName}},
	)

	if err != nil {
		log.Printf("Failed to update category: %v", err)
		http.Error(w, "Failed to update category", http.StatusInternalServerError)
		return
	}

	// Get updated category
	var updatedCategory models.PantryCategory
	err = config.DB.Collection("pantry_categories").FindOne(
		context.Background(),
		bson.M{"_id": categoryID},
	).Decode(&updatedCategory)

	if err != nil {
		log.Printf("Failed to fetch updated category: %v", err)
		http.Error(w, "Failed to fetch updated category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CategoryResponse{
		Status:  "success",
		Message: "Category updated successfully",
		Data:    updatedCategory,
	})
}

// DeletePantryCategoryHandler deletes a custom category (only if no items use it)
func DeletePantryCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from context (set by AuthMiddleware)
	userClaims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Get category ID from URL path
	categoryIDStr := strings.TrimPrefix(r.URL.Path, "/api/pantry/categories/")
	if categoryIDStr == "" {
		http.Error(w, "Category ID is required", http.StatusBadRequest)
		return
	}

	// Convert category ID to ObjectID
	categoryID, err := primitive.ObjectIDFromHex(categoryIDStr)
	if err != nil {
		http.Error(w, "Invalid category ID format", http.StatusBadRequest)
		return
	}

	// Get user ID
	userID, err := primitive.ObjectIDFromHex(userClaims.ID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Find user to get their group
	var user models.User
	err = config.DB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		}
		return
	}

	// Find the category to delete
	var category models.PantryCategory
	err = config.DB.Collection("pantry_categories").FindOne(
		context.Background(),
		bson.M{"_id": categoryID},
	).Decode(&category)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Category not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch category", http.StatusInternalServerError)
		}
		return
	}

	// Check permissions FIRST
	if !category.CanBeDeletedBy(userID, user.GroupID) {
		if category.IsPredefined() {
			http.Error(w, "Predefined categories cannot be deleted", http.StatusForbidden)
		} else {
			http.Error(w, "You can only delete custom categories from your group", http.StatusForbidden)
		}
		return
	}

	// Check if any pantry items are using this category by ID
	itemCount, err := config.DB.Collection("pantry_items").CountDocuments(
		context.Background(),
		bson.M{"category_id": categoryID},
	)

	if err != nil {
		log.Printf("Failed to check pantry items using category: %v", err)
		http.Error(w, "Failed to check category usage", http.StatusInternalServerError)
		return
	}

	if itemCount > 0 {
		http.Error(w, "Cannot delete category: it is being used by pantry items", http.StatusConflict)
		return
	}

	// Delete the category
	result, err := config.DB.Collection("pantry_categories").DeleteOne(
		context.Background(),
		bson.M{"_id": categoryID},
	)

	if err != nil {
		log.Printf("Failed to delete category: %v", err)
		http.Error(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "Category not found or already deleted", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CategoryResponse{
		Status:  "success",
		Message: "Category deleted successfully",
	})
}
