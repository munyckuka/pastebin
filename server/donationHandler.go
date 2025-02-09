package server

import (
	"html/template"
	"log"
	"net/http"
)

func DonationHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/payment.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		log.Println("Ошибка рендеринга страницы:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
	}
}
