package main

import (
	"fmt"
	"net/http"
	"pastebin/cmd/server"
)

func main() {
	server.ConnectToDB()
	http.HandleFunc("/", server.MainPageHandler)
	http.HandleFunc("/create-paste", server.CreatePasteHandler)
	server.GetCollection("pastes")
	server.ConnectToDB()
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
