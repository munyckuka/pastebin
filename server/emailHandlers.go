package server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
)

// Обработчик для отправки email
func SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Получаем данные из формы
	email := r.FormValue("email")
	message := r.FormValue("message")

	if email == "" || message == "" {
		http.Error(w, "Email and message fields cannot be empty", http.StatusBadRequest)
		return
	}

	// Отправка письма
	err := sendEmail(email, message)
	if err != nil {
		http.Error(w, "Failed to send email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Успешный ответ
	fmt.Fprintf(w, "Email successfully sent to %s", email)
}

// Функция для отправки письма
func sendEmail(to string, body string) error {
	// Настройки SMTP
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	senderEmail := "ofblooms@gmail.com"     // Укажите вашу почту
	senderPassword := "qewa htwv qwoc xbrf" // Укажите ваш пароль

	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpHost)

	// Форматирование письма
	msg := []byte("To: " + to + "\r\n" +
		"Subject: Admin Notification\r\n" +
		"\r\n" +
		body + "\r\n")

	// Отправка
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{to}, msg)
}

func sendEmailWithAttachment(to string, subject string, body string, attachmentPath string) error {
	// SMTP настройки
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	senderEmail := "your-email@gmail.com"
	senderPassword := "your-email-password"

	// Создаем буфер для составления письма
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Заголовки письма
	headers := map[string]string{
		"From":         senderEmail,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()),
	}
	for key, value := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	buf.WriteString("\r\n")

	// Тело письма
	textWriter, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"text/plain; charset=\"utf-8\""},
	})
	if err != nil {
		return err
	}
	textWriter.Write([]byte(body))

	// Вложение файла
	if attachmentPath != "" {
		file, err := os.Open(attachmentPath)
		if err != nil {
			return fmt.Errorf("cannot open file: %v", err)
		}
		defer file.Close()

		filePart, err := writer.CreatePart(map[string][]string{
			"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, attachmentPath)},
			"Content-Type":              {"application/octet-stream"},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return err
		}

		// Читаем файл и кодируем его в Base64
		fileContent := make([]byte, 0)
		buffer := make([]byte, 512)
		for {
			n, err := file.Read(buffer)
			if err != nil {
				break
			}
			fileContent = append(fileContent, buffer[:n]...)
		}
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(fileContent)))
		base64.StdEncoding.Encode(encoded, fileContent)
		filePart.Write(encoded)
	}

	// Закрываем writer
	writer.Close()

	// Аутентификация и отправка письма
	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpHost)
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{to}, buf.Bytes())
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	log.Println("Email sent successfully to:", to)
	return nil
}
