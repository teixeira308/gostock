package middleware

import (
	"context"
	"gostock/internal/pkg/cache"
	"net"
	"net/http"
	"strconv"
	"time"
)

func RateLimiter(client cache.Client, limit int, duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			key := "rate-limit:" + ip
			ctx := context.Background()

			count, err := client.GetInt(ctx, key)
			if err == cache.ErrCacheMiss {
				client.Set(ctx, key, 1, duration)
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-1))
				next.ServeHTTP(w, r)
				return
			} else if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if count >= limit {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			client.Incr(ctx, key)
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-count-1))
			next.ServeHTTP(w, r)
		})
	}
}
