package middleware

import (
	"log"
	"net/http"
	"time"
)

func JSONMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		start := time.Now()
		log.Printf("[%s] %s - %s", r.Method, r.RequestURI, time.Since(start))
		next(w, r)
	}
}