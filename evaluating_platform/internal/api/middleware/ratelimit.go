package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu     sync.Mutex
	counts map[string][]time.Time
	limit  int
	window time.Duration
}

// RateLimit 简单滑动窗口限流中间件
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		counts: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
	return func(c *gin.Context) {
		key := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			key = userID.(string)
		}

		if !rl.allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 清理过期记录
	times := rl.counts[key]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.counts[key] = valid
		return false
	}

	rl.counts[key] = append(valid, now)
	return true
}
