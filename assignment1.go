package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Message struct {
	Message string `json:"message"`
}

func HandlePostRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only Post requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	var message Message

	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		response := map[string]string{
			"status":  "fail",
			"message": "Invalid json message",
		}
		w.Header().Set("Contenet-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	if message.Message == "" {
		response := map[string]string{
			"status":  "fail",
			"message": "field `message` is required",
		}
		w.Header().Set("Contenet-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	fmt.Printf("Recieved message: %s\n", message.Message)

	response := map[string]string{
		"status":  "success",
		"message": "data successfully received",
	}
	w.Header().Set("Contenet-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return
}

func main() {

	http.HandleFunc("/post", HandlePostRequest)
	fmt.Println("Server: http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))

}
