package server

import (
	"context"
	"log"

	"fmt"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"pastebin/models"
)

// Путь к шаблонам
const templatesDir = "web"

// Главная страница
func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles(templatesDir + "/home.html"))
	tmpl.Execute(w, nil)
}

// Страница создания пасты
func CreatePasteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r.ParseForm()
	content := r.FormValue("content")
	title := r.FormValue("title")

	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	collection := GetCollection("pastes")

	// Создание объекта пасты
	paste := models.Paste{
		ID:        primitive.NewObjectID(),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	_, err := collection.InsertOne(ctx, paste)
	if err != nil {
		http.Error(w, "Failed to save paste", http.StatusInternalServerError)
		return
	}

	// test
	log.Printf("Создана паста с ID: %s", paste.ID.Hex())

	// Перенаправление на страницу пасты
	http.Redirect(w, r, fmt.Sprintf("/paste/%s", paste.ID.Hex()), http.StatusSeeOther)
}

// Просмотр пасты по ID
func ViewPasteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Конвертируем строку ID в ObjectId
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Неверный формат ID", http.StatusBadRequest)
		return
	}

	collection := GetCollection("pastes")
	var paste models.Paste
	err = collection.FindOne(r.Context(), bson.M{"_id": objID}).Decode(&paste)
	if err == mongo.ErrNoDocuments {
		http.Error(w, "Паста не найдена", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	// Счетчик кол-во просмотров
	_, err = collection.UpdateOne(
		r.Context(),
		bson.M{"_id": objID},
		bson.M{"$inc": bson.M{"current_reads": 1}},
	)
	if err != nil {
		log.Printf("Ошибка при обновлении счётчика просмотров: %v", err)
		// Не прерываем выполнение, так как это не критично
	}

	// Отображаем страницу с данными пасты
	tmpl := template.Must(template.ParseFiles("web/readpaste.html"))
	err = tmpl.Execute(w, paste)
	if err != nil {
		http.Error(w, "Ошибка рендеринга шаблона", http.StatusInternalServerError)
	}
}

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Отображаем страницу регистрации
		tmpl, err := template.ParseFiles("web/signup.html")
		if err != nil {
			http.Error(w, "Failed to load signup page", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		// Создаем контекст для запроса
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Парсим данные из формы
		login := r.FormValue("login")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if login == "" || email == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Проверка длины пароля
		if len(password) < 8 {
			http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
			return
		}

		// Подключаемся к коллекции пользователей
		collection := GetCollection("users")

		// Проверяем, существует ли уже пользователь с таким email
		var existingUser models.User
		err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)
		if err == nil {
			// Пользователь с таким email уже существует
			http.Error(w, "Email is already registered", http.StatusConflict)
			return
		} else if err != mongo.ErrNoDocuments {
			// Ошибка при попытке найти пользователя
			http.Error(w, "Failed to check existing user", http.StatusInternalServerError)
			return
		}

		// Хэшируем пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Создаем нового пользователя
		user := models.User{
			ID:       primitive.NewObjectID(),
			Login:    login,
			Email:    email,
			Password: string(hashedPassword),
		}

		// Сохраняем пользователя в базе данных
		_, err = collection.InsertOne(ctx, user)
		if err != nil {
			http.Error(w, "Failed to save user", http.StatusInternalServerError)
			return
		}

		// Ответ при успешной регистрации
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if r.Method == http.MethodGet {
		tmpl, _ := template.ParseFiles("web/login.html")
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		email := r.FormValue("email")
		password := r.FormValue("password")

		var user models.User
		collection := GetCollection("users")
		err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		http.Redirect(w, r, "/account", http.StatusSeeOther)
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
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Заглушка для проверки пользователя
	login := "Guest" // Извлечение логина пользователя из сессии

	// Получаем новый пароль
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

	// Обновляем пароль в базе данных
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("users")
	_, err = collection.UpdateOne(ctx, bson.M{"login": login}, bson.M{"$set": bson.M{"password": string(hashedPassword)}})
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/account", http.StatusSeeOther)
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
