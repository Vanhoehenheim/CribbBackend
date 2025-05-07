package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username    string             `bson:"username" json:"username"`
	Password    string             `bson:"password" json:"-"`
	Name        string             `bson:"name" json:"name"`
	PhoneNumber string             `bson:"phone_number" json:"phone_number"`
	RoomNumber  string             `bson:"room_number" json:"room_number"`
	Score       int                `bson:"score" json:"score"`
	Group       string             `bson:"group" json:"group"`
	GroupID     primitive.ObjectID `bson:"group_id" json:"group_id"`
	GroupCode   string             `bson:"group_code" json:"group_code"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
