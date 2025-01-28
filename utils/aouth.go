package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"net/http"
	"net/smtp"
	"os"
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

var (
	googleOauthConfig *oauth2.Config
	oauthStateString  = "randomstatestring" // Используется для защиты от CSRF
)

func init() {
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"), // Задайте через переменные окружения
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/oauth/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state != oauthStateString {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получение данных о пользователе
	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Логика обработки (например, создание пользователя в базе)
	fmt.Fprintf(w, "User Info: %s (%s)", userInfo.Name, userInfo.Email)
}
