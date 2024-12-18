package server

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

// Главная страница
func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/home.html")
}

// CreatePasteHandler обрабатывает создание пасты
func CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Структура для входных данных
	var paste struct {
		Expires     string `json:"expires"`
		Content     string `json:"content"`
		Title       string `json:"title"`
		Password    string `json:"password"`
		DeleteAfter int    `json:"deleteAfter"`
	}

	err := json.NewDecoder(r.Body).Decode(&paste)
	if err != nil || paste.Expires == "" || paste.Content == "" {
		http.Error(w, `{"message": "Field 'expires' and 'content' are required"}`, http.StatusBadRequest)
		return
	}

	// Получаем коллекцию "pastes"
	collection := GetCollection("pastes")

	// Формируем документ для вставки
	newPaste := bson.M{
		"expires":      paste.Expires,
		"content":      paste.Content,
		"title":        paste.Title,
		"password":     paste.Password,
		"deleteAfter":  paste.DeleteAfter,
		"currentReads": 0,
		"createdAt":    time.Now(),
	}

	// Вставляем документ в коллекцию
	_, err = collection.InsertOne(context.Background(), newPaste)
	if err != nil {
		http.Error(w, `{"message": "Failed to save paste"}`, http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Paste successfully created!",
	})
}
