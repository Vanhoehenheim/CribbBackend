package handlers

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"fmt"     // For formatted I/O
	"strings" // For string manipulation
	"time"    // For time-related operations

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options" // MongoDB options
)

// CreateGroupHandler creates a new group
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var group models.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Initialize with proper defaults (including group_code generation)
	group = *models.NewGroup(group.Name)

	// Insert and get generated ID
	result, err := config.DB.Collection("groups").InsertOne(context.Background(), group)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			http.Error(w, "Group name already exists", http.StatusConflict)
		} else {
			log.Printf("Group creation error: %v", err)
			http.Error(w, "Failed to create group", http.StatusInternalServerError)
		}
		return
	}

	// Set generated ID from MongoDB
	group.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

type JoinGroupRequest struct {
	Username   string `json:"username"`
	GroupName  string `json:"group_name"`
	GroupCode  string `json:"groupCode"`
	RoomNumber string `json:"roomNo"`
}

func JoinGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Determine how to find the group - by name or by code
	var groupFilter bson.M
	if request.GroupName != "" {
		groupFilter = bson.M{"name": request.GroupName}
	} else if request.GroupCode != "" {
		groupFilter = bson.M{"group_code": request.GroupCode}
	} else {
		http.Error(w, "Either group_name or groupCode is required", http.StatusBadRequest)
		return
	}

	// Start MongoDB session
	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Session start error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	// Transaction handling
	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// 1. Fetch group with essential fields
		var group models.Group
		err := config.DB.Collection("groups").FindOne(
			sc,
			groupFilter,
			options.FindOne().SetProjection(bson.M{"name": 1, "group_code": 1}),
		).Decode(&group)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("group not found")
			}
			log.Printf("Group fetch error: %v", err)
			return fmt.Errorf("failed to fetch group")
		}

		// 2. Fetch user with essential fields
		var user models.User
		err = config.DB.Collection("users").FindOne(
			sc,
			bson.M{"username": request.Username},
			options.FindOne().SetProjection(bson.M{"_id": 1}),
		).Decode(&user)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("user not found")
			}
			log.Printf("User fetch error: %v", err)
			return fmt.Errorf("failed to fetch user")
		}

		// 3. Update user document with room number if provided
		updateFields := bson.M{
			"group":      group.Name,
			"group_id":   group.ID,
			"group_code": group.GroupCode,
			"updated_at": time.Now(),
		}

		if request.RoomNumber != "" {
			updateFields["room_number"] = request.RoomNumber
		}

		userUpdate := bson.M{
			"$set": updateFields,
		}

		userRes, err := config.DB.Collection("users").UpdateByID(
			sc,
			user.ID,
			userUpdate,
			options.Update().SetUpsert(false),
		)
		if err != nil {
			log.Printf("User update error: %v", err)
			return fmt.Errorf("failed to update user group")
		}
		if userRes.MatchedCount == 0 {
			return fmt.Errorf("user document not found")
		}

		// 4. Update group members array
		groupUpdate := bson.M{
			"$addToSet": bson.M{"members": user.ID},
			"$set":      bson.M{"updated_at": time.Now()},
		}
		groupRes, err := config.DB.Collection("groups").UpdateByID(
			sc,
			group.ID,
			groupUpdate,
		)
		if err != nil {
			log.Printf("Group members update error: %v", err)
			return fmt.Errorf("failed to update group members: %v", err)
		}
		if groupRes.MatchedCount == 0 {
			return fmt.Errorf("group document not found")
		}

		return nil
	})

	// Handle transaction result
	if err != nil {
		log.Printf("Transaction failed: %v", err)
		switch {
		case strings.Contains(err.Error(), "group not found"):
			http.Error(w, "Group not found", http.StatusNotFound)
		case strings.Contains(err.Error(), "user not found"):
			http.Error(w, "User not found", http.StatusNotFound)
		case strings.Contains(err.Error(), "user document not found"):
			http.Error(w, "User document not found", http.StatusNotFound)
		case strings.Contains(err.Error(), "group document not found"):
			http.Error(w, "Group document not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully joined group",
	})
}

// GetGroupMembersHandler retrieves all members of a group by group name
func GetGroupMembersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupIdentifier := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	var filter bson.M
	if groupIdentifier != "" {
		filter = bson.M{"name": groupIdentifier}
	} else if groupCode != "" {
		filter = bson.M{"group_code": groupCode}
	} else {
		http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
		return
	}

	// Fetch the group by name or code
	var group models.Group
	err := config.DB.Collection("groups").FindOne(context.Background(), filter).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Fetch all users in the group
	cursor, err := config.DB.Collection("users").Find(context.Background(), bson.M{"group_id": group.ID})
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		http.Error(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GroupDetailsResponse defines the response structure for group metadata
// It purposely omits the members array to keep the payload small.
type GroupDetailsResponse struct {
	ID          primitive.ObjectID `json:"id"`
	Name        string             `json:"name"`
	GroupCode   string             `json:"group_code"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	MemberCount int64              `json:"member_count"`
	TotalPoints int64              `json:"total_points"`
}

// GetGroupDetailsHandler returns high-level metadata about a group (name, code, member count, total points, etc.)
func GetGroupDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	var filter bson.M
	if groupName != "" {
		filter = bson.M{"name": groupName}
	} else if groupCode != "" {
		filter = bson.M{"group_code": groupCode}
	} else {
		http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
		return
	}

	// Fetch group document
	var group models.Group
	if err := config.DB.Collection("groups").FindOne(ctx, filter).Decode(&group); err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			log.Printf("GetGroupDetailsHandler find error: %v", err)
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Count members using users collection (safer than len(group.Members) in case of stale data)
	memberCount, err := config.DB.Collection("users").CountDocuments(ctx, bson.M{"group_id": group.ID})
	if err != nil {
		log.Printf("GetGroupDetailsHandler count members error: %v", err)
		http.Error(w, "Failed to count members", http.StatusInternalServerError)
		return
	}

	// Aggregate total points for users in this group
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"group_id": group.ID}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$score"}}}},
	}
	cursor, err := config.DB.Collection("users").Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("GetGroupDetailsHandler aggregate error: %v", err)
		http.Error(w, "Failed to aggregate points", http.StatusInternalServerError)
		return
	}
	var aggResult []bson.M
	if err := cursor.All(ctx, &aggResult); err != nil {
		log.Printf("GetGroupDetailsHandler cursor decode error: %v", err)
		http.Error(w, "Failed to aggregate points", http.StatusInternalServerError)
		return
	}
	var totalPoints int64
	if len(aggResult) > 0 {
		if v, ok := aggResult[0]["total"].(int32); ok {
			totalPoints = int64(v)
		} else if v, ok := aggResult[0]["total"].(int64); ok {
			totalPoints = v
		}
	}

	response := GroupDetailsResponse{
		ID:          group.ID,
		Name:        group.Name,
		GroupCode:   group.GroupCode,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
		MemberCount: memberCount,
		TotalPoints: totalPoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetGroupLeaderboardHandler returns members of a group sorted by score in descending order
func GetGroupLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	groupName := r.URL.Query().Get("group_name")
	groupCode := r.URL.Query().Get("group_code")

	var filter bson.M
	if groupName != "" {
		filter = bson.M{"name": groupName}
	} else if groupCode != "" {
		filter = bson.M{"group_code": groupCode}
	} else {
		http.Error(w, "Either group_name or group_code is required", http.StatusBadRequest)
		return
	}

	// Fetch group document to obtain its ID
	var group models.Group
	if err := config.DB.Collection("groups").FindOne(ctx, filter).Decode(&group); err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Group not found", http.StatusNotFound)
		} else {
			log.Printf("GetGroupLeaderboardHandler find error: %v", err)
			http.Error(w, "Failed to fetch group", http.StatusInternalServerError)
		}
		return
	}

	// Retrieve users for the group sorted by score DESC
	opts := options.Find().SetSort(bson.D{{Key: "score", Value: -1}})
	cursor, err := config.DB.Collection("users").Find(ctx, bson.M{"group_id": group.ID}, opts)
	if err != nil {
		log.Printf("GetGroupLeaderboardHandler find users error: %v", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		log.Printf("GetGroupLeaderboardHandler cursor decode error: %v", err)
		http.Error(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// ============================
// Leave Group Handler (POST)
// Endpoint: /api/groups/leave
// Request Body: { "username": "alice", "carryForward": true }
// ============================
type LeaveGroupRequest struct {
	Username     string `json:"username"`
	CarryForward bool   `json:"carryForward"`
}

func LeaveGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		// Handle pre-flight CORS quickly
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// CORS header (in case global middleware didn't fire)
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
	w.Header().Set("Content-Type", "application/json")

	var req LeaveGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	session, err := config.DB.Client().StartSession()
	if err != nil {
		log.Printf("Session start error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer session.EndSession(context.Background())

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		// 1. Fetch user document
		var user models.User
		if err := config.DB.Collection("users").FindOne(sc, bson.M{"username": req.Username}).Decode(&user); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("user not found")
			}
			return fmt.Errorf("failed to fetch user: %v", err)
		}

		// If user not in a group, nothing to do
		if user.GroupID.IsZero() {
			return fmt.Errorf("user is not in a group")
		}

		// 2. Update group document: pull member
		_, err := config.DB.Collection("groups").UpdateByID(sc, user.GroupID, bson.M{
			"$pull": bson.M{"members": user.ID},
			"$set":  bson.M{"updated_at": time.Now()},
		})
		if err != nil {
			return fmt.Errorf("failed to update group: %v", err)
		}

		// 3. Update user document: unset group fields, optionally reset score
		update := bson.M{
			"$unset": bson.M{
				"group":       "",
				"group_id":    "",
				"group_code":  "",
				"room_number": "",
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}

		if !req.CarryForward {
			update["$set"].(bson.M)["score"] = 0
		}

		if _, err := config.DB.Collection("users").UpdateByID(sc, user.ID, update); err != nil {
			return fmt.Errorf("failed to update user: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Leave group error: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(bson.M{"message": "left group successfully"})
}
