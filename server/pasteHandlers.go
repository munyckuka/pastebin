package server

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"strconv"

	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"pastebin/models"
)

// logger
var pasteLogger *log.Logger

func InitLogger() {
	file, err := os.OpenFile("logs/paste_actions.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Создаем логгер
	pasteLogger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	log.Println("Paste logger initialized.")
}

func HandleError(w http.ResponseWriter, err error, statusCode int, message string) {
	// Логгирование ошибки
	if err != nil {
		pasteLogger.Printf("Error: %v | StatusCode: %d | Message: %s", err, statusCode, message)
	} else {
		pasteLogger.Printf("StatusCode: %d | Message: %s", statusCode, message)
	}

	// Отправка ответа клиенту
	http.Error(w, message, statusCode)
}

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

	// Установка таймаута для контекста
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Парсинг данных формы
	err := r.ParseForm()
	if err != nil {
		HandleError(w, err, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	// Проверка обязательных полей
	if content == "" {
		HandleError(w, nil, http.StatusBadRequest, "Content is required")
		return
	}

	// Получаем коллекцию из MongoDB
	collection := GetCollection("pastes")

	// Создание объекта пасты
	paste := models.Paste{
		ID:        primitive.NewObjectID(),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Сохранение пасты в базе данных
	_, err = collection.InsertOne(ctx, paste)
	if err != nil {
		log.Printf("Ошибка сохранения пасты: %v", err)
		HandleError(w, err, http.StatusInternalServerError, "Failed to save paste")
		return
	}

	// Логируем создание пасты
	if pasteLogger != nil {
		pasteLogger.Printf("Created paste: ID=%s, Title=%s, Date=%s\n", paste.ID.Hex(), paste.Title, paste.CreatedAt.Format(time.RFC3339))
	} else {
		log.Printf("Логгер не инициализирован: создана паста с ID=%s", paste.ID.Hex())
	}

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
		HandleError(w, err, http.StatusInternalServerError, "Invalid ID format")
		return
	}

	collection := GetCollection("pastes")
	var paste models.Paste
	err = collection.FindOne(r.Context(), bson.M{"_id": objID}).Decode(&paste)
	if err == mongo.ErrNoDocuments {
		HandleError(w, err, http.StatusInternalServerError, "Paste not found")
		return
	} else if err != nil {
		HandleError(w, err, http.StatusInternalServerError, "BD connection error")
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

func AllPastesHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем параметры сортировки, фильтрации и пагинации из URL
	sortOrder, _ := strconv.Atoi(r.URL.Query().Get("sort"))
	if sortOrder != 1 && sortOrder != -1 {
		sortOrder = 1 // Значение по умолчанию
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1 // Первая страница по умолчанию
	}

	limit := 5 // Количество паст на одной странице
	skip := (page - 1) * limit

	filter := r.URL.Query().Get("filter")
	now := time.Now()
	var filterCondition bson.M

	switch filter {
	case "last-year":
		filterCondition = bson.M{"createdAt": bson.M{"$gte": now.AddDate(-1, 0, 0)}}
	case "last-month":
		filterCondition = bson.M{"createdAt": bson.M{"$gte": now.AddDate(0, -1, 0)}}
	case "last-week":
		filterCondition = bson.M{"createdAt": bson.M{"$gte": now.AddDate(0, 0, -7)}}
	case "last-day":
		filterCondition = bson.M{"createdAt": bson.M{"$gte": now.AddDate(0, 0, -1)}}
	default:
		filterCondition = bson.M{} // Без фильтрации
	}

	// Настройки сортировки, лимита и пропуска
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "createdAt", Value: sortOrder}})
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(skip))

	collection := GetCollection("pastes")
	cursor, err := collection.Find(ctx, filterCondition, findOptions)
	if err != nil {
		log.Printf("Error fetching pastes: %v\n", err)
		http.Error(w, "Failed to fetch pastes", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var pastes []models.Paste
	if err := cursor.All(ctx, &pastes); err != nil {
		log.Printf("Error decoding pastes: %v\n", err)
		http.Error(w, "Failed to decode pastes", http.StatusInternalServerError)
		return
	}

	// Рендеринг шаблона
	tmpl := template.Must(template.ParseFiles("web/allpastes.html"))
	data := struct {
		Pastes []models.Paste
		Page   int
		Next   int
		Prev   int
	}{
		Pastes: pastes,
		Page:   page,
		Next:   page + 1,
		Prev:   page - 1,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error rendering template: %v\n", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func DeletePasteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Получение ID из URL
	//vars := mux.Vars(r)
	id := "asd"

	// Конвертация строки ID в ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("Invalid paste ID format: %s", id)
		HandleError(w, err, http.StatusBadRequest, "Invalid paste ID")
		return
	}

	// Удаление из базы данных
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("pastes")
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		log.Printf("Failed to delete paste with ID %s: %v", id, err)
		HandleError(w, err, http.StatusInternalServerError, "Failed to delete paste")
		return
	}

	// Проверка, была ли паста удалена
	if result.DeletedCount == 0 {
		log.Printf("No paste found with ID %s to delete", id)
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	// Логирование удаления
	if pasteLogger != nil {
		pasteLogger.Printf("Deleted paste: ID=%s, Date=%s\n", objID.Hex(), time.Now().Format(time.RFC3339))
	} else {
		log.Printf("Логгер не инициализирован: удалена паста с ID=%s", objID.Hex())
	}

	// Редирект на список паст
	http.Redirect(w, r, "/all-pastes", http.StatusSeeOther)
}

func EditPasteHandler(w http.ResponseWriter, r *http.Request) {
	// Получение ID из URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Конвертируем строку ID в ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		HandleError(w, err, http.StatusBadRequest, "Invalid paste ID")
		return
	}

	// Получаем коллекцию
	collection := GetCollection("pastes")

	if r.Method == http.MethodGet {
		// Получаем пасту из базы
		var paste models.Paste
		err := collection.FindOne(r.Context(), bson.M{"_id": objID}).Decode(&paste)
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Paste not found", http.StatusNotFound)
			return
		} else if err != nil {
			pasteLogger.Printf("Database error while fetching paste ID=%s: %v\n", id, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Отображаем форму редактирования
		tmpl := template.Must(template.ParseFiles("web/editpaste.html"))
		if err := tmpl.Execute(w, paste); err != nil {
			pasteLogger.Printf("Template execution error for paste ID=%s: %v\n", id, err)
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
		return
	}

	if r.Method == http.MethodPost {
		// Получаем данные из формы
		r.ParseForm()
		title := r.FormValue("title")
		content := r.FormValue("content")

		// Обновляем данные в базе
		update := bson.M{
			"$set": bson.M{
				"title":   title,
				"content": content,
			},
		}
		_, err := collection.UpdateOne(r.Context(), bson.M{"_id": objID}, update)
		if err != nil {
			pasteLogger.Printf("Database update error for paste ID=%s: %v\n", id, err)
			http.Error(w, "Failed to update paste", http.StatusInternalServerError)
			return
		}

		// Логгирование
		pasteLogger.Printf("Edited paste: ID=%s, Date=%s\n", id, time.Now().Format(time.RFC3339))

		// Редирект на страницу просмотра пасты
		http.Redirect(w, r, fmt.Sprintf("/paste/%s", id), http.StatusSeeOther)
		return
	}

	// Метод не поддерживается
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
