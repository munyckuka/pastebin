package main

import (
	"context"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pastebin/middleware"
	"pastebin/server"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func setupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Rate limiting
	r.Use(middleware.RateLimiterMiddleware)

	// Определите маршруты
	// r.Handle("/", middleware.AuthMiddleware(http.HandlerFunc(server.MainPageHandler))).Methods("GET")
	r.HandleFunc("/", server.MainPageHandler).Methods("GET")
	r.HandleFunc("/create-paste", server.CreatePasteHandler).Methods("POST")
	r.HandleFunc("/paste/{id}", server.ViewPasteHandler).Methods("GET")
	r.HandleFunc("/admin", middleware.AdminMiddleware(server.AllPastesHandler)).Methods("GET")

	r.HandleFunc("/pastes/{id}/delete", server.DeletePasteHandler).Methods("POST")
	r.HandleFunc("/pastes/{id}/edit", server.EditPasteHandler).Methods("GET", "POST")

	r.HandleFunc("/signup", server.SignupHandler).Methods("GET", "POST")
	r.HandleFunc("/login", server.LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/profile", server.ProfileHandler)
	r.HandleFunc("/logout", server.LogoutHandler).Methods("POST")

	r.HandleFunc("/oauth/google", server.GoogleLoginHandler)
	r.HandleFunc("/oauth/google/callback", server.GoogleCallbackHandler)

	r.HandleFunc("/send-email", server.SendEmailHandler).Methods("POST")
	r.HandleFunc("/verify-email/{token}", server.VerifyEmailHandler).Methods("GET")

	return r
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}
	server.InitGoogleOAuth()
	// Подключаемся к базе данных
	err1 := server.ConnectToDB()
	if err1 != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err1)
	}

	// Инициализация логгера
	server.InitLogger()

	// Создаем сервер
	srv := &http.Server{
		Addr:    ":8080",
		Handler: setupRoutes(),
	}

	// Канал для получения сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Println("Starting server on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-quit
	log.Println("Server is shutting down...")

	// Контекст для завершения операций
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Закрываем сервер
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Закрываем базу данных
	log.Println("Closing database connections...")
	if err := server.CloseDB(); err != nil {
		log.Printf("Error closing database: %v", err)
	} else {
		log.Println("Database connections closed.")
	}

	log.Println("Server exited gracefully")
}
