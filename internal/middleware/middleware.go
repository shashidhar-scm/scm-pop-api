package middleware

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	window  time.Duration
	max     int
	mu      sync.Mutex
	clients map[string]*rateEntry
}

type rateEntry struct {
	count   int
	expires time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	if max <= 0 || window <= 0 {
		return nil
	}
	return &rateLimiter{
		window:  window,
		max:     max,
		clients: make(map[string]*rateEntry),
	}
}

func (l *rateLimiter) allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.clients[key]
	if !ok || now.After(entry.expires) {
		l.clients[key] = &rateEntry{count: 1, expires: now.Add(l.window)}
		return true
	}

	if entry.count >= l.max {
		return false
	}

	entry.count++
	return true
}

func RateLimit(next http.Handler) http.Handler {
	window := time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second
	max := getEnvInt("RATE_LIMIT_MAX", 120)
	limiter := newRateLimiter(max, window)
	if limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(clientKey(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientKey(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func getEnvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
