package server

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database // Global variable to hold the database instance

// ConnectToDB establishes a connection to the MongoDB database
func ConnectToDB() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI("mongodb+srv://kuka:<1234>@pastebin.2ojuf.mongodb.net/?retryWrites=true&w=majority&appName=PasteBin").SetServerAPIOptions(serverAPI)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Assign the database to the global variable
	db = client.Database("pastebin")
	log.Println("Connected to MongoDB!")
}

// CloseDB closes the connection to MongoDB
func CloseDB() {
	if db != nil {
		log.Println("Closing database connection...")
		if err := db.Client().Disconnect(context.Background()); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}
}

// GetCollection returns a collection from the database
func GetCollection(collectionName string) *mongo.Collection {
	if db == nil {
		log.Fatal("Database connection is not initialized. Call ConnectToDB() first.")
	}
	return db.Collection(collectionName)
}
