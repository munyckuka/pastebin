package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"pastebin/server"
)

func main() {
	// Подключаемся к базе данных
	err := server.ConnectToDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	// Настраиваем маршрутизатор
	r := mux.NewRouter()
	r.HandleFunc("/", server.MainPageHandler).Methods("GET")
	r.HandleFunc("/create-paste", server.CreatePasteHandler).Methods("POST")
	r.HandleFunc("/paste/{id}", server.ViewPasteHandler).Methods("GET")
	r.HandleFunc("/signup", server.SignupHandler).Methods("GET", "POST")
	r.HandleFunc("/login", server.LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/users", server.UsersHandler).Methods("GET")
	r.HandleFunc("/delete-user/{id}", server.DeleteUserHandler).Methods("POST")

	// Запускаем сервер
	fmt.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
