// middleware/access_control.go
package middleware

import (
	"context"
	"cribb-backend/config"
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupAccessControlMiddleware ensures that users can only access resources from their own group
func GroupAccessControlMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context (set by AuthMiddleware)
		userClaims, ok := GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get group ID from query parameters (if any)
		groupName := r.URL.Query().Get("group_name")
		groupCode := r.URL.Query().Get("group_code")

		// If neither group name nor code provided, let the handler handle it
		if groupName == "" && groupCode == "" {
			next(w, r)
			return
		}

		// Get user ID
		userID, err := primitive.ObjectIDFromHex(userClaims.ID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Find user to get their group
		var user struct {
			GroupID primitive.ObjectID `bson:"group_id"`
		}
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

		// Find the requested group
		var groupFilter bson.M
		if groupName != "" {
			groupFilter = bson.M{"name": groupName}
		} else {
			groupFilter = bson.M{"group_code": groupCode}
		}

		var group struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		err = config.DB.Collection("groups").FindOne(
			context.Background(),
			groupFilter,
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

		// Store the verified group ID in context for easy access
		ctx := context.WithValue(r.Context(), "verified_group_id", group.ID)
		next(w, r.WithContext(ctx))
	}
}

// GetVerifiedGroupID retrieves the verified group ID from the request context
func GetVerifiedGroupID(ctx context.Context) (primitive.ObjectID, bool) {
	groupID, ok := ctx.Value("verified_group_id").(primitive.ObjectID)
	return groupID, ok
}

// ResourceOwnershipMiddleware ensures that users can only modify resources they own
func ResourceOwnershipMiddleware(next http.HandlerFunc, resourceCollection string, resourceIDParam string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For GET requests, we don't need to check ownership
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}

		// Get user from context (set by AuthMiddleware)
		userClaims, ok := GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get user ID
		userID, err := primitive.ObjectIDFromHex(userClaims.ID)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Get resource ID from URL or query parameter
		var resourceIDStr string
		if resourceIDParam == "path" {
			// Extract from URL path
			path := r.URL.Path
			segments := strings.Split(path, "/")
			if len(segments) > 0 {
				resourceIDStr = segments[len(segments)-1]
			}
		} else {
			// Extract from query parameter
			resourceIDStr = r.URL.Query().Get(resourceIDParam)
			if resourceIDStr == "" {
				// If not in query, try to extract from request body
				// Note: This would require reading the body, which might interfere with subsequent processing
				// For simplicty, we'll assume the ID is in the URL or query parameters
			}
		}

		// If no resource ID found, let the handler handle it
		if resourceIDStr == "" {
			next(w, r)
			return
		}

		// Convert to ObjectID
		resourceID, err := primitive.ObjectIDFromHex(resourceIDStr)
		if err != nil {
			http.Error(w, "Invalid resource ID format", http.StatusBadRequest)
			return
		}

		// Find the resource to check ownership
		var resource struct {
			UserID primitive.ObjectID `bson:"user_id"`
		}
		err = config.DB.Collection(resourceCollection).FindOne(
			context.Background(),
			bson.M{"_id": resourceID},
		).Decode(&resource)

		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				http.Error(w, "Resource not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch resource", http.StatusInternalServerError)
			}
			return
		}

		// Verify user owns the resource
		if resource.UserID != userID {
			http.Error(w, "You do not have permission to modify this resource", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
