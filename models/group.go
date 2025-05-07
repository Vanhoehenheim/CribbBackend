package models

import (
	"context"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Group struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name      string               `bson:"name" json:"name" validate:"required,min=3"`
	GroupCode string               `bson:"group_code" json:"group_code"`
	Members   []primitive.ObjectID `bson:"members" json:"members"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}

func generateGroupCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, 6)
	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}
	return string(code)
}

func NewGroup(name string) *Group {
	return &Group{
		Name:      name,
		GroupCode: generateGroupCode(),
		Members:   make([]primitive.ObjectID, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// MigrateExistingGroups adds group codes to existing groups
func MigrateExistingGroups(db *mongo.Database) error {
	ctx := context.Background()
	cursor, err := db.Collection("groups").Find(
		ctx,
		bson.M{"group_code": bson.M{"$exists": false}},
	)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err = cursor.All(ctx, &groups); err != nil {
		return err
	}

	for _, group := range groups {
		_, err = db.Collection("groups").UpdateOne(
			ctx,
			bson.M{"_id": group.ID},
			bson.M{"$set": bson.M{"group_code": generateGroupCode()}},
		)
		if err != nil {
			return err
		}
	}
	return nil
}
