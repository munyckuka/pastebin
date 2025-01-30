package utils

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"net/http"
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
func GenerateToken(userID primitive.ObjectID, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.Hex(), // записываем user_id в строковом формате
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
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
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", jwt.NewValidationError("invalid token claims", jwt.ValidationErrorClaimsInvalid)
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", jwt.NewValidationError("email not found in token claims", jwt.ValidationErrorClaimsInvalid)
	}

	return email, nil
}

func GetUserIDFromToken(r *http.Request) (primitive.ObjectID, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("token not found")
	}

	tokenString := cookie.Value
	claims := &jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid token")
	}

	// Получаем user_id из токена
	userIDStr, ok := (*claims)["user_id"].(string)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("invalid token claims")
	}

	// Конвертируем строковый user_id в ObjectID
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid user ID format")
	}

	return userID, nil
}
