package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
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

// renderSignupPage отображает страницу регистрации с сообщением об ошибке
func renderSignupPage(w http.ResponseWriter, errorMessage, login, email string) {
	tmpl, err := template.ParseFiles("web/signup.html")
	if err != nil {
		http.Error(w, "Failed to load signup page", http.StatusInternalServerError)
		return
	}

	// Передаем сообщение об ошибке и текущие данные в шаблон
	data := map[string]interface{}{
		"ErrorMessage": errorMessage,
		"Login":        login,
		"Email":        email,
	}
	tmpl.Execute(w, data)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Проверка пользователя
		var user models.User
		err := db.Collection("users").FindOne(context.TODO(), bson.M{"email": email}).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Сравнение пароля
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Проверка, подтвержден ли email
		if !user.IsVerified {
			http.Error(w, "Email not verified", http.StatusForbidden)
			return
		}

		// Успешная авторизация (создание сессии или JWT)
		fmt.Fprintf(w, "Login successful!")
	} else {
		// Отображаем форму входа
		tmpl := template.Must(template.ParseFiles("web/login.html"))
		tmpl.Execute(w, nil)
	}
}
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	collection := GetCollection("users")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		http.Error(w, "Error decoding users", http.StatusInternalServerError)
		return
	}

	tmpl, _ := template.ParseFiles("web/users.html")
	tmpl.Execute(w, users)
}
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := r.URL.Path[len("/delete-user/"):]
	objID, _ := primitive.ObjectIDFromHex(id)

	collection := GetCollection("users")
	_, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
func AccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Заглушка для проверки текущего пользователя
	// В реальной реализации используйте сессию или токен для идентификации пользователя
	email := "example@example.com" // Временно, заменить на реальный email из сессии

	collection := GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		// Загрузка страницы аккаунта
		tmpl, err := template.ParseFiles("web/account.html")
		if err != nil {
			http.Error(w, "Failed to load account page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, user)
		return
	}

	if r.Method == http.MethodPost {
		// Обновление данных пользователя
		newPassword := r.FormValue("password")

		if newPassword != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			_, err = collection.UpdateOne(ctx, bson.M{"email": email}, bson.M{"$set": bson.M{"password": string(hashedPassword)}})
			if err != nil {
				http.Error(w, "Failed to update password", http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем user-id из URL
	vars := mux.Vars(r)
	userID := vars["user-id"]

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Заглушка для проверки пользователя
	// На реальном проекте здесь будет извлечение логина или ID из сессии
	// Например, userID может быть получен из сессии или токена
	// login := getLoginFromSession(r)  // Тестовый логин

	// Получаем новый пароль из формы
	newPassword := r.FormValue("new-password")
	if len(newPassword) < 8 {
		http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
		return
	}

	// Хэшируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Подключаемся к базе данных
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("users")
	// Обновляем пароль пользователя, чей ID совпадает с user-id
	_, err = collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"password": string(hashedPassword)}})
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Перенаправляем на страницу аккаунта
	http.Redirect(w, r, "/account/"+userID, http.StatusSeeOther)
}

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Временно: используйте сессию или токен для идентификации пользователя
	email := "example@example.com"

	collection := GetCollection("users")
	_, err := collection.DeleteOne(ctx, bson.M{"email": email})
	if err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	// Редирект на главную страницу после удаления
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
