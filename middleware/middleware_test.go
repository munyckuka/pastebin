package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"pastebin/models"
	"pastebin/server"
	"strings"
	"testing"
	"time"
)

func TestRateLimiterMiddleware(t *testing.T) {
	// Тестовый обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Применение middleware
	testHandler := RateLimiterMiddleware(handler)

	// Создание тестового сервера
	server := httptest.NewServer(testHandler)
	defer server.Close()

	client := server.Client()

	// Отправка запросов
	for i := 0; i < 10; i++ { // Больше, чем requestsPerSecond
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if i < 5 && resp.StatusCode != http.StatusOK { // первые 5 запросов успешны
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if i >= 5 && resp.StatusCode != http.StatusTooManyRequests { // остальные заблокированы
			t.Errorf("Expected status 429, got %d", resp.StatusCode)
		}
	}
}

func TestAdminMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		expectCode int
	}{
		{"Без токена", "", http.StatusForbidden},
		{"Недействительный токен", "invalid_token", http.StatusUnauthorized},
		{"Не-админ", generateToken("user@example.com"), http.StatusForbidden},
		{"Админ", generateToken("admin@example.com"), http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin", nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{Name: "token", Value: tt.token})
			}

			rec := httptest.NewRecorder()
			handler := AdminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"message": "success"})
			}))

			handler.ServeHTTP(rec, req)
			if rec.Code != tt.expectCode {
				t.Errorf("Ожидался статус %d, получен %d", tt.expectCode, rec.Code)
			}
		})
	}
}

// Поддельная база данных
var mockUsers = map[string]models.User{
	"admin@example.com": {Email: "admin@example.com", Role: "admin"},
	"user@example.com":  {Email: "user@example.com", Role: "user"},
}

// Поддельная функция поиска пользователя
func mockFindUserByEmail(ctx context.Context, email string) (models.User, error) {
	user, exists := mockUsers[email]
	if !exists {
		return models.User{}, errors.New("user not found")
	}
	return user, nil
}

// Функция для генерации JWT-токена
func generateToken(email string) string {
	claims := jwt.MapClaims{
		"email": email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte("your-secret-key"))
	return signedToken
}

// Глобальная переменная для тестовой БД
var testDB *mongo.Database

// Настройка маршрутов (замена SetupRouter)
func setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/login", server.LoginHandler).Methods("POST")
	r.HandleFunc("/admin", AdminMiddleware(server.AllPastesHandler)).Methods("GET")
	return r
}

// Настройка тестовой MongoDB
func setupTestDB() *mongo.Database {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Ошибка подключения к MongoDB: %v", err)
	}

	// Проверим подключение
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatalf("MongoDB не отвечает: %v", err)
	}

	log.Println("✅ Успешное подключение к тестовой БД")
	return client.Database("testdb")
}

// Тестовые данные
var testAdmin = models.User{
	Email:        "ofblooms@gmail.com",
	PasswordHash: "1234",
	Role:         "admin",
}

// 🔹 **E2E-тест**
func TestEndToEnd(t *testing.T) {
	// Инициализируем БД и проверяем, что не nil
	testDB = setupTestDB()
	if testDB == nil {
		t.Fatal("❌ testDB не была инициализирована!")
	}

	// Создаём тестового админа
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := testDB.Collection("users").InsertOne(ctx, testAdmin)
	if err != nil {
		t.Fatalf("❌ Ошибка при создании тестового пользователя: %v", err)
	}

	// Запускаем сервер
	server := httptest.NewServer(setupRouter())
	defer server.Close()

	// HTTP-клиент с куками
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Логин запрос
	loginData := `{"email": "admin@example.com", "password": "1234"}`
	resp, err := client.Post(server.URL+"/login", "application/json", strings.NewReader(loginData))
	if err != nil {
		t.Fatalf("❌ Ошибка при отправке запроса на /login: %v", err)
	}

	// Проверяем статус код
	assert.Equal(t, http.StatusOK, resp.StatusCode, "❌ Ошибка авторизации")

	log.Println("✅ Авторизация прошла успешно")

	// Проверяем наличие кук
	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies, "❌ Кука с токеном отсутствует")
}
