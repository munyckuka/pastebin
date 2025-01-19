package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"pastebin/middleware" // замените на ваш реальный путь
	"testing"
)

func TestRateLimiterMiddleware(t *testing.T) {
	// Тестовый обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Применение middleware
	testHandler := middleware.RateLimiterMiddleware(handler)

	// Создание тестового сервера
	server := httptest.NewServer(testHandler)
	defer server.Close()

	client := server.Client()

	// Отправка запросов
	for i := 0; i < 10; i++ { // Больше, чем requestsPerSecond
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if i < 5 && resp.StatusCode != http.StatusOK { // первые 5 запросов успешны
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if i >= 5 && resp.StatusCode != http.StatusTooManyRequests { // остальные заблокированы
			t.Errorf("Expected status 429, got %d", resp.StatusCode)
		}
	}
}
