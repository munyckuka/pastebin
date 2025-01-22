package email

import (
	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "admin@example.com") // Замените на ваш email
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body) // HTML-формат

	d := gomail.NewDialer("smtp.example.com", 587, "admin@example.com", "yourpassword") // Замените на ваши настройки

	return d.DialAndSend(m)
}
