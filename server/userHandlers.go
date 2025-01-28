package server

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
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

		// Проверка на совпадение паролей
		if password != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		// Проверка на наличие пользователя с таким email
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
			IsVerified:   false, // Не подтверждено
		}

		// Сохранение пользователя в базу данных
		_, err = db.Collection("users").InsertOne(context.TODO(), user)
		if err != nil {
			http.Error(w, "Error saving user", http.StatusInternalServerError)
			return
		}

		// Генерация токена для подтверждения email
		token := utils.GenerateToken(email)
		// Отправка письма с ссылкой для подтверждения email
		err = utils.SendVerificationEmail(email, token)
		if err != nil {
			http.Error(w, "Failed to send verification email", http.StatusInternalServerError)
			return
		}

		// Ответ пользователю
		fmt.Fprintf(w, "Registration successful. Please verify your email.")
	} else {
		// Отображаем форму регистрации
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
			Email        string `bson:"email"`
			PasswordHash string `bson:"password_hash"`
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
		token := utils.GenerateToken(email)
		// Установка токена в cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
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
	// Извлекаем токен из cookie
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Token not found", http.StatusUnauthorized)
		return
	}

	// Декодируем токен
	tokenString := cookie.Value
	claims := &jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Получаем email из токена
	email, ok := (*claims)["email"].(string)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	// Ищем пользователя в базе данных
	var user struct {
		Email string `bson:"email"`
	}
	err = db.Collection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Отображаем профиль пользователя
	fmt.Fprintf(w, "Welcome to your profile, %s!", user.Email)
}
