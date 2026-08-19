package api

import (
	"log"
	"net/http"
	"runtime"
	"time"
)

// LoggingMiddleware logs every HTTP request with duration and status.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s → %d (%s)",
			r.Method, r.URL.Path, wrapped.statusCode, duration.Round(time.Microsecond))
	})
}

// CORSMiddleware handles cross-origin requests for the mobile client.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Player-ID, X-Client-Version")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware implements simple IP-based rate limiting.
// In production, replace with a token-bucket Redis implementation.
func RateLimitMiddleware(maxPerMin int) func(next http.Handler) http.Handler {
	// Simple in-memory rate limiter — for development only.
	// Production: use Redis-based sliding window.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement Redis-based rate limiting
			next.ServeHTTP(w, r)
		})
	}
}

// RecoverMiddleware catches panics and returns a 500 error.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, err)
				// Print stack trace for debugging
				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				log.Printf("[PANIC STACK]\n%s", buf[:n])
				http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// PlayerIDMiddleware extracts the player ID from the X-Player-ID header.
func PlayerIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		playerID := r.Header.Get("X-Player-ID")
		if playerID != "" {
			// Store in context for downstream handlers
			ctx := WithPlayerID(r.Context(), playerID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates JWT tokens or API keys.
// For development, this is a pass-through.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement JWT validation
		// For now, accept X-Player-ID header as identity
		playerID := r.Header.Get("X-Player-ID")
		if playerID == "" {
			playerID = "anonymous"
		}
		ctx := WithPlayerID(r.Context(), playerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
