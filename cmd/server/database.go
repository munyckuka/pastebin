package server

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database // Global variable to hold the database instance

// ConnectToDB establishes a connection to the MongoDB database
func ConnectToDB() {
	// Replace the placeholder with your Atlas connection string
	const uri = "mongodb+srv://kuka:1234@pastebin.2ojuf.mongodb.net/?retryWrites=true&w=majority&appName=PasteBin"

	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping the database to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Assign the database to the global variable
	db = client.Database("pastebin")
	log.Println("Connected to MongoDB!")
}

// CloseDB closes the connection to MongoDB
func CloseDB() {
	if db != nil {
		log.Println("Closing database connection...")
		if err := db.Client().Disconnect(context.TODO()); err != nil {
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
