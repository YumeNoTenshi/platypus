package api

import (
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Вызов следующего обработчика
		next.ServeHTTP(w, r)

		// Логирование после обработки запроса
		log.Printf(
			"%s %s %s %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			time.Since(start),
		)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Key")
		
		// В реальном приложении здесь должна быть проверка токена
		if token == "" {
			respondWithError(w, http.StatusUnauthorized, "No API key provided")
			return
		}

		// Продолжаем выполнение
		next.ServeHTTP(w, r)
	})
} 