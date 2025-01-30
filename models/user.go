package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email"`
	Name         string             `bson:"Name"`
	PasswordHash string             `bson:"password_hash"`
	IsVerified   bool               `bson:"is_verified"`
	Provider     string             `bson:"provider,omitempty"` // Добавил поле для OAuth
}
