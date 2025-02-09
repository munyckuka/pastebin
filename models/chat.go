package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Chat struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	AdminID   primitive.ObjectID `bson:"admin_id,omitempty"`
	Status    string             `bson:"status"` // "active" / "inactive"
	Messages  []Message          `bson:"messages"`
	CreatedAt time.Time          `bson:"created_at"`
}

type Message struct {
	Sender    string    `bson:"sender"` // "user" или "admin"
	Content   string    `bson:"content"`
	Timestamp time.Time `bson:"timestamp"`
}
