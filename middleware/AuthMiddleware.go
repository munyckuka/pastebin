package middleware

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

var jwtSecret = []byte("cAtwa1kkEy")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		// Извлекаем токен из заголовка
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Извлекаем email из токена
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		email, ok := claims["email"].(string)
		if !ok {
			http.Error(w, "Email not found in token", http.StatusUnauthorized)
			return
		}

		// Передаём email в контекст
		ctx := context.WithValue(r.Context(), "userEmail", email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
