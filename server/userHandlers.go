package server

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"pastebin/models"
	"pastebin/utils"
	"time"
)

// Путь к шаблонам
const templatesDir = "web"

var jwtSecret = []byte("cAtwa1kkEy")

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		if password != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		// Проверка, существует ли пользователь
		var existingUser models.User
		err := db.Collection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&existingUser)
		if err == nil {
			http.Error(w, "Email already registered", http.StatusBadRequest)
			return
		}

		// Хеширование пароля
		passwordHash, err := utils.HashPassword(password)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}

		// Создание нового пользователя
		user := models.User{
			Email:        email,
			PasswordHash: passwordHash,
			IsVerified:   false,
		}

		// Вставка пользователя в базу данных
		result, err := db.Collection("users").InsertOne(context.TODO(), user)
		if err != nil {
			http.Error(w, "Error saving user", http.StatusInternalServerError)
			return
		}

		// Получаем `userID` из результата вставки
		userID := result.InsertedID.(primitive.ObjectID)

		// Генерация токена с `userID`
		token := utils.GenerateToken(userID, email)

		// Отправляем email для подтверждения
		err = utils.SendVerificationEmail(email, token)
		if err != nil {
			http.Error(w, "Failed to send verification email", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Registration successful. Please verify your email.")
	} else {
		tmpl := template.Must(template.ParseFiles("web/signup.html"))
		tmpl.Execute(w, nil)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Получаем данные из формы
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Проверяем пользователя в базе данных
		var user struct {
			ID           primitive.ObjectID `bson:"_id"`
			Email        string             `bson:"email"`
			PasswordHash string             `bson:"password_hash"`
		}
		err := db.Collection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Проверяем пароль
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Генерация токена
		token := utils.GenerateToken(user.ID, email)
		// Установка токена в cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		http.Redirect(w, r, "/profile", http.StatusSeeOther)

	} else {
		// Если метод GET, отображаем форму логина
		tmpl := template.Must(template.ParseFiles("web/login.html"))
		tmpl.Execute(w, nil)
	}
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем токен из куки
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Token not found", http.StatusUnauthorized)
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

	// Ищем пользователя по `_id`
	var user struct {
		Name  string `bson:"name"`
		Email string `bson:"email"`
	}
	err = db.Collection("users").FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Получаем все paste-записи пользователя
	var pastes []models.Paste
	cursor, err := db.Collection("pastes").Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		http.Error(w, "Failed to fetch pastes", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var paste models.Paste
		if err := cursor.Decode(&paste); err != nil {
			http.Error(w, "Error decoding paste", http.StatusInternalServerError)
			return
		}
		pastes = append(pastes, paste)
	}

	// Загружаем HTML-шаблон
	tmpl := template.Must(template.ParseFiles("web/profile.html"))
	tmpl.Execute(w, struct {
		Name   string         `json:"name"`
		Email  string         `json:"email"`
		Pastes []models.Paste `json:"pastes"`
	}{
		Name:   user.Name,
		Email:  user.Email,
		Pastes: pastes,
	})
}
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Удаляем куку с токеном
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Устанавливаем истёкшую дату
		HttpOnly: true,
	})

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusOK)
}
