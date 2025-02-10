package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"html/template"
	"log"
	"net/http"
	"pastebin/models"
	"pastebin/utils"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketMessage struct {
	ChatID  string `json:"chat_id"`
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

func handleWebSocket(conn *websocket.Conn, chatID string) {
	defer conn.Close()

	for {
		var msg WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			// Проверяем, не был ли это нормальный разрыв соединения
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Ошибка чтения JSON:", err)
			} else {
				//log.Println("Клиент отключился:", err)
			}
			break
		}

		// Сохранение сообщения в БД
		message := models.Message{
			Sender:    msg.Sender,
			Content:   msg.Content,
			Timestamp: time.Now(),
		}
		chatObjectID, _ := primitive.ObjectIDFromHex(chatID)
		_, err = GetCollection("chats").UpdateOne(
			context.TODO(),
			bson.M{"_id": chatObjectID},
			bson.M{"$push": bson.M{"messages": message}},
		)
		if err != nil {
			log.Println("Ошибка сохранения сообщения:", err)
			continue
		}

		// Отправляем сообщение обратно всем подключённым клиентам
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Println("Ошибка отправки JSON:", err)
			break
		}
		//log.Printf("Получено сообщение: %+v\n", msg)

	}
}

func ChatWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket подключен!")
	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка WebSocket соединения:", err)
		return
	}
	//log.Println("Клиент подключился к чату:", chatID)

	// Запускаем обработчик
	handleWebSocket(conn, chatID)

	//log.Println("Клиент отключился от чата:", chatID)
}

func CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем токен из куки
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	}

	// Декодируем токен
	tokenString := cookie.Value
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Получаем userID из токена
	userIDHex, ok := (*claims)["user_id"].(string)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	// Преобразуем строку userID в ObjectID для MongoDB
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	existingChat := GetCollection("chats").FindOne(context.TODO(), bson.M{"user_id": userID, "status": "active"})

	if existingChat.Err() == nil {
		http.Error(w, "У вас уже есть активный чат", http.StatusBadRequest)
		return
	}

	newChat := models.Chat{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Status:    "active",
		Messages:  []models.Message{},
		CreatedAt: time.Now(),
	}

	_, err = GetCollection("chats").InsertOne(context.TODO(), newChat)
	if err != nil {
		http.Error(w, "Ошибка создания чата", http.StatusInternalServerError)
		return
	}
	redirectURL := fmt.Sprintf("/chat/%s", newChat.ID.Hex())
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
func getActiveChatsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, err := GetCollection("chats").Find(context.TODO(), bson.M{"status": "active"})
	if err != nil {
		http.Error(w, "Ошибка получения чатов", http.StatusInternalServerError)
		return
	}
	var chats []models.Chat
	if err := cursor.All(context.TODO(), &chats); err != nil {
		http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(chats)
}
func CloseChatHandler(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")
	chatObjectID, _ := primitive.ObjectIDFromHex(chatID)

	_, err := GetCollection("chats").UpdateOne(
		context.TODO(),
		bson.M{"_id": chatObjectID},
		bson.M{"$set": bson.M{"status": "inactive", "messages": []models.Message{}}},
	)
	if err != nil {
		http.Error(w, "Ошибка закрытия чата", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func ChatPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, ok := vars["id"]
	if !ok {
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	// Получаем userID из токена
	userID, err := utils.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Подключаемся к базе данных
	var user struct {
		Role string `bson:"role"`
	}

	// Проверяем роль пользователя
	err = db.Collection("users").FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Error(w, "Ошибка получения пользователя", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("web/chat.html")
	if err != nil {
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Передаём данные в шаблон
	data := struct {
		ChatID  string
		IsAdmin bool
	}{
		ChatID:  chatID,
		IsAdmin: user.Role == "admin", // Проверяем, является ли пользователь админом
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
	}
}

func GetChatHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	chatID := r.URL.Query().Get("chat_id")

	if chatID == "" {
		http.Error(w, `{"error": "chat_id обязателен"}`, http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(chatID)
	if err != nil {
		http.Error(w, `{"error": "Неверный формат chat_id"}`, http.StatusBadRequest)
		return
	}

	var chat struct {
		Messages []struct {
			Sender    string    `json:"sender"`
			Content   string    `json:"content"`
			Timestamp time.Time `json:"timestamp"` // Исправили на time.Time
		} `json:"messages"`
	}

	err = db.Collection("chats").
		FindOne(context.TODO(), bson.M{"_id": objectID}).
		Decode(&chat)

	if err == mongo.ErrNoDocuments {
		http.Error(w, `{"error": "Чат не найден"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println("Ошибка получения чата:", err)
		http.Error(w, `{"error": "Ошибка загрузки сообщений"}`, http.StatusInternalServerError)
		return
	}

	// Преобразуем time.Time в строку перед отправкой
	var response struct {
		Messages []struct {
			Sender    string `json:"sender"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		} `json:"messages"`
	}

	for _, msg := range chat.Messages {
		response.Messages = append(response.Messages, struct {
			Sender    string `json:"sender"`
			Content   string `json:"content"`
			Timestamp string `json:"timestamp"`
		}{
			Sender:    msg.Sender,
			Content:   msg.Content,
			Timestamp: msg.Timestamp.Format(time.RFC3339), // Форматируем дату
		})
	}

	json.NewEncoder(w).Encode(response.Messages)
}
func AllChatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем список всех чатов
	collection := GetCollection("chats")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Ошибка получения чатов: %v\n", err)
		http.Error(w, "Ошибка получения чатов", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var chats []models.Chat
	if err := cursor.All(ctx, &chats); err != nil {
		log.Printf("Ошибка декодирования чатов: %v\n", err)
		http.Error(w, "Ошибка декодирования чатов", http.StatusInternalServerError)
		return
	}

	// Рендерим страницу
	tmpl := template.Must(template.ParseFiles("web/allchats.html"))
	err = tmpl.Execute(w, chats)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v\n", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
	}
}
