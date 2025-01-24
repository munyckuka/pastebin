package utils

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"net/smtp"
	"time"
)

var jwtSecret = []byte("cAtwa1kkEy")

// Функция хеширования пароля
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

// Функция для генерации токена
func GenerateToken(email string) string {
	// Используем JWT для создания токена
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

// Функция отправки email с ссылкой для подтверждения
func SendVerificationEmail(email, token string) error {
	// Генерация ссылки для подтверждения
	verificationLink := fmt.Sprintf("http://localhost:8080/verify-email/%s", token)

	// Настройки SMTP-сервера
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	senderEmail := "ofblooms@gmail.com"
	senderPassword := "qewa htwv qwoc xbrf"

	// Текст сообщения
	subject := "Subject: Email Verification\n"
	body := fmt.Sprintf("Please click the following link to verify your email: %s", verificationLink)
	message := subject + "\n" + body

	// Отправка письма
	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{email}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	fmt.Println("Verification email sent to:", email)
	return nil
}
func DecodeToken(tokenString string) (string, error) {
	// Парсим токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Убедимся, что метод подписи совпадает
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	// Проверяем валидность токена
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Извлекаем email из токена
		email, ok := claims["email"].(string)
		if !ok {
			return "", errors.New("email not found in token")
		}
		return email, nil
	}
	return "", errors.New("invalid token")
}
