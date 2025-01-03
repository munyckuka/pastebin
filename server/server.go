package server

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var db *mongo.Database

// Подключение к MongoDB
func ConnectToDB() error {
	clientOptions := options.Client().ApplyURI("mongodb+srv://kuka:1234@pastebin.2ojuf.mongodb.net/?retryWrites=true&w=majority&appName=PasteBin")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}

	// Проверяем подключение
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}

	// Устанавливаем базу данных
	db = client.Database("pastebin")
	log.Println("Успешное подключение к MongoDB")
	return nil
}

// Получить коллекцию из базы данных
func GetCollection(name string) *mongo.Collection {
	return db.Collection(name)
}
