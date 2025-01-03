package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Структура для пасты
type Paste struct {
	ID           primitive.ObjectID `bson:"_id"`
	Title        string             `bson:"title"`
	Content      string             `bson:"content"`
	CreatedAt    time.Time          `bson:"createdAt"`
	Expires      string             `bson:"expires"`
	Password     string             `bson:"password"`
	DeleteAfter  int32              `bson:"deleteAfter"`
	CurrentReads int32              `bson:"currentReads"`
}
