// jobs/pantry_jobs.go
package jobs

import (
	"context"
	"cribb-backend/config"
	"cribb-backend/models"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// StartPantryJobs initializes and starts the pantry background jobs
func StartPantryJobs() {
	log.Println("Starting pantry background jobs...")

	// Run jobs every 6 hours
	ticker := time.NewTicker(6 * time.Hour)

	// Run immediately once at startup
	go checkExpiringItems()
	go checkLowStockItems()

	// Then run on the schedule
	go func() {
		for range ticker.C {
			checkExpiringItems()
			checkLowStockItems()
		}
	}()
}

// checkExpiringItems looks for items that will expire soon and creates notifications
func checkExpiringItems() {
	log.Println("Checking for expiring pantry items...")

	// Find items that will expire in the next 3 days
	now := time.Now()
	expirationThreshold := now.AddDate(0, 0, 3)

	// Find items that will expire soon but haven't been marked yet
	// (no existing notification of type expiring_soon)
	cursor, err := config.DB.Collection("pantry_items").Find(
		context.Background(),
		bson.M{
			"expiration_date": bson.M{
				"$gte": now,
				"$lte": expirationThreshold,
			},
		},
	)

	if err != nil {
		log.Printf("Error finding expiring items: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var expiringItems []models.PantryItem
	if err = cursor.All(context.Background(), &expiringItems); err != nil {
		log.Printf("Error decoding expiring items: %v", err)
		return
	}

	// Process each item and create notifications if needed
	for _, item := range expiringItems {
		// Check if a notification already exists for this item
		count, err := config.DB.Collection("pantry_notifications").CountDocuments(
			context.Background(),
			bson.M{
				"item_id": item.ID,
				"type":    models.NotificationTypeExpiringSoon,
				"created_at": bson.M{
					"$gte": now.AddDate(0, 0, -3), // Only check for notifications in the last 3 days
				},
			},
		)

		if err != nil {
			log.Printf("Error checking existing notifications: %v", err)
			continue
		}

		// If no notification exists, create one
		if count == 0 {
			notification := models.CreatePantryNotification(
				item.GroupID,
				item.ID,
				item.Name,
				models.NotificationTypeExpiringSoon,
				"Item will expire in 3 days or less",
			)

			_, err = config.DB.Collection("pantry_notifications").InsertOne(
				context.Background(),
				notification,
			)

			if err != nil {
				log.Printf("Error creating expiration notification: %v", err)
			} else {
				log.Printf("Created expiration notification for item: %s", item.Name)
			}
		}
	}

	// Also check for already expired items
	cursor, err = config.DB.Collection("pantry_items").Find(
		context.Background(),
		bson.M{
			"expiration_date": bson.M{
				"$lt": now,
			},
		},
	)

	if err != nil {
		log.Printf("Error finding expired items: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var expiredItems []models.PantryItem
	if err = cursor.All(context.Background(), &expiredItems); err != nil {
		log.Printf("Error decoding expired items: %v", err)
		return
	}

	// Process each expired item
	for _, item := range expiredItems {
		// Check if a notification already exists for this item
		count, err := config.DB.Collection("pantry_notifications").CountDocuments(
			context.Background(),
			bson.M{
				"item_id": item.ID,
				"type":    models.NotificationTypeExpired,
				"created_at": bson.M{
					"$gte": now.AddDate(0, 0, -3), // Only check for notifications in the last 3 days
				},
			},
		)

		if err != nil {
			log.Printf("Error checking existing notifications: %v", err)
			continue
		}

		// If no notification exists, create one
		if count == 0 {
			notification := models.CreatePantryNotification(
				item.GroupID,
				item.ID,
				item.Name,
				models.NotificationTypeExpired,
				"Item has expired",
			)

			_, err = config.DB.Collection("pantry_notifications").InsertOne(
				context.Background(),
				notification,
			)

			if err != nil {
				log.Printf("Error creating expired notification: %v", err)
			} else {
				log.Printf("Created expired notification for item: %s", item.Name)
			}
		}
	}

	log.Printf("Completed expiring items check, found %d expiring and %d expired items",
		len(expiringItems), len(expiredItems))
}

// checkLowStockItems looks for items that are running low and creates notifications
func checkLowStockItems() {
	log.Println("Checking for low stock and out of stock pantry items...")
	now := time.Now()

	// First handle out of stock items
	cursor, err := config.DB.Collection("pantry_items").Find(
		context.Background(),
		bson.M{
			"quantity": 0,
		},
	)

	if err != nil {
		log.Printf("Error finding out of stock items: %v", err)
	} else {
		defer cursor.Close(context.Background())

		var outOfStockItems []models.PantryItem
		if err = cursor.All(context.Background(), &outOfStockItems); err != nil {
			log.Printf("Error decoding out of stock items: %v", err)
		} else {
			// Process each out of stock item
			for _, item := range outOfStockItems {
				// Check if a notification already exists for this item
				count, err := config.DB.Collection("pantry_notifications").CountDocuments(
					context.Background(),
					bson.M{
						"item_id": item.ID,
						"type":    models.NotificationTypeOutOfStock,
						"created_at": bson.M{
							"$gte": now.AddDate(0, 0, -3), // Only check for notifications in the last 3 days
						},
					},
				)

				if err != nil {
					log.Printf("Error checking existing notifications: %v", err)
					continue
				}

				// If no notification exists, create one
				if count == 0 {
					// First delete any existing low stock notifications for this item
					_, err := config.DB.Collection("pantry_notifications").DeleteMany(
						context.Background(),
						bson.M{
							"item_id": item.ID,
							"type":    models.NotificationTypeLowStock,
						},
					)

					if err != nil {
						log.Printf("Error deleting low stock notifications: %v", err)
					}

					// Then create the out of stock notification
					notification := models.CreatePantryNotification(
						item.GroupID,
						item.ID,
						item.Name,
						models.NotificationTypeOutOfStock,
						"Item is out of stock",
					)

					_, err = config.DB.Collection("pantry_notifications").InsertOne(
						context.Background(),
						notification,
					)

					if err != nil {
						log.Printf("Error creating out of stock notification: %v", err)
					} else {
						log.Printf("Created out of stock notification for item: %s", item.Name)
					}
				}
			}

			log.Printf("Completed out of stock check, found %d items", len(outOfStockItems))
		}
	}

	// Then handle low stock items (but exclude items with quantity 0)
	lowStockThreshold := 1.0 // Setting a fixed threshold for simplicity

	cursor, err = config.DB.Collection("pantry_items").Find(
		context.Background(),
		bson.M{
			"quantity": bson.M{
				"$gt":  0,
				"$lte": lowStockThreshold,
			},
		},
	)

	if err != nil {
		log.Printf("Error finding low stock items: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var lowStockItems []models.PantryItem
	if err = cursor.All(context.Background(), &lowStockItems); err != nil {
		log.Printf("Error decoding low stock items: %v", err)
		return
	}

	// Process each low stock item
	for _, item := range lowStockItems {
		// Check if a notification already exists for this item
		count, err := config.DB.Collection("pantry_notifications").CountDocuments(
			context.Background(),
			bson.M{
				"item_id": item.ID,
				"type":    models.NotificationTypeLowStock,
				"created_at": bson.M{
					"$gte": now.AddDate(0, 0, -3), // Only check for notifications in the last 3 days
				},
			},
		)

		if err != nil {
			log.Printf("Error checking existing notifications: %v", err)
			continue
		}

		// If no notification exists, create one
		if count == 0 {
			notification := models.CreatePantryNotification(
				item.GroupID,
				item.ID,
				item.Name,
				models.NotificationTypeLowStock,
				"Item is running low",
			)

			_, err = config.DB.Collection("pantry_notifications").InsertOne(
				context.Background(),
				notification,
			)

			if err != nil {
				log.Printf("Error creating low stock notification: %v", err)
			} else {
				log.Printf("Created low stock notification for item: %s", item.Name)
			}
		}
	}

	log.Printf("Completed low stock check, found %d items", len(lowStockItems))
}

// GenerateShoppingList automatically creates a shopping list based on low stock items
func GenerateShoppingList(groupID primitive.ObjectID) ([]map[string]interface{}, error) {
	// Find all low stock items
	cursor, err := config.DB.Collection("pantry_items").Find(
		context.Background(),
		bson.M{
			"group_id": groupID,
			"quantity": bson.M{
				"$lte": 1.0, // Low stock threshold
			},
		},
	)

	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var items []models.PantryItem
	if err = cursor.All(context.Background(), &items); err != nil {
		return nil, err
	}

	// Also include items with notifications of type low_stock
	notifCursor, err := config.DB.Collection("pantry_notifications").Find(
		context.Background(),
		bson.M{
			"group_id": groupID,
			"type":     models.NotificationTypeLowStock,
			"created_at": bson.M{
				"$gte": time.Now().AddDate(0, 0, -7), // Consider notifications from the last week
			},
		},
	)

	if err != nil {
		return nil, err
	}
	defer notifCursor.Close(context.Background())

	var notifications []models.PantryNotification
	if err = notifCursor.All(context.Background(), &notifications); err != nil {
		return nil, err
	}

	// Create a map to track items already in the shopping list
	itemMap := make(map[string]bool)
	shoppingList := make([]map[string]interface{}, 0)

	// Add low stock items to the shopping list
	for _, item := range items {
		if !itemMap[item.ID.Hex()] {
			itemMap[item.ID.Hex()] = true

			// Suggest a quantity to buy based on typical usage
			suggestedQuantity := 1.0
			if item.Quantity <= 0 {
				suggestedQuantity = 2.0 // If completely out, suggest buying 2
			}

			shoppingList = append(shoppingList, map[string]interface{}{
				"item_id":            item.ID.Hex(),
				"name":               item.Name,
				"category":           item.Category,
				"current_quantity":   item.Quantity,
				"unit":               item.Unit,
				"suggested_quantity": suggestedQuantity,
				"reason":             "Low stock",
			})
		}
	}

	// Add items from low stock notifications if not already in the list
	for _, notification := range notifications {
		if !itemMap[notification.ItemID.Hex()] {
			// Fetch the current item data
			var item models.PantryItem
			err := config.DB.Collection("pantry_items").FindOne(
				context.Background(),
				bson.M{"_id": notification.ItemID},
			).Decode(&item)

			if err != nil {
				if err != mongo.ErrNoDocuments {
					log.Printf("Error fetching item %s: %v", notification.ItemID.Hex(), err)
				}
				continue
			}

			itemMap[notification.ItemID.Hex()] = true

			// Suggest a quantity based on typical usage
			suggestedQuantity := 2.0 // Default suggestion

			shoppingList = append(shoppingList, map[string]interface{}{
				"item_id":            item.ID.Hex(),
				"name":               item.Name,
				"category":           item.Category,
				"current_quantity":   item.Quantity,
				"unit":               item.Unit,
				"suggested_quantity": suggestedQuantity,
				"reason":             notification.Message,
			})
		}
	}

	return shoppingList, nil
}
