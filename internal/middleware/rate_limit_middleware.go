package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	Count     int
	ExpiresAt time.Time
}

type RateLimiter struct {
	mutex sync.Mutex

	entries map[string]rateLimitEntry

	limit int

	window time.Duration
}

func NewRateLimiter(
	limit int,
	window time.Duration,
) *RateLimiter {
	return &RateLimiter{
		entries: make(
			map[string]rateLimitEntry,
		),
		limit:  limit,
		window: window,
	}
}

func (r *RateLimiter) LimitByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now().UTC()

		r.mutex.Lock()

		entry, exists := r.entries[key]

		if !exists ||
			now.After(entry.ExpiresAt) {
			entry = rateLimitEntry{
				Count:     0,
				ExpiresAt: now.Add(r.window),
			}
		}

		entry.Count++
		r.entries[key] = entry

		allowed := entry.Count <= r.limit

		r.mutex.Unlock()

		if !allowed {
			response.Error(
				c,
				http.StatusTooManyRequests,
				"Too many login attempts",
				nil,
			)

			c.Abort()
			return
		}

		c.Next()
	}
}