package middleware

import (
	"api/metrics"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
    visitors map[string]*Visitor
    mu       sync.Mutex
    rate     int           // Maximum requests per minute
    burst    int           // Burst capacity
    interval time.Duration // Refill interval
}

type Visitor struct {
    tokens      int
    lastUpdated time.Time
}

func NewRateLimiter(rate int, burst int) *RateLimiter {
    return &RateLimiter{
        visitors: make(map[string]*Visitor),
        rate:     rate,
        burst:    burst,
        interval: time.Minute,
    }
}

func (rl *RateLimiter) getVisitor(ip string) *Visitor {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    if visitor, exists := rl.visitors[ip]; exists {
        return visitor
    }

    visitor := &Visitor{
        tokens:      rl.burst,
        lastUpdated: time.Now(),
    }
    rl.visitors[ip] = visitor
    return visitor
}

func (rl *RateLimiter) Allow(ip string) bool {
    visitor := rl.getVisitor(ip)

    rl.mu.Lock()
    defer rl.mu.Unlock()

    // Refill tokens
    now := time.Now()
    elapsed := now.Sub(visitor.lastUpdated)
    refill := int(elapsed / rl.interval)
    if refill > 0 {
        visitor.tokens += refill * rl.rate
        if visitor.tokens > rl.burst {
            visitor.tokens = rl.burst
        }
        visitor.lastUpdated = now
    }

    // Check if request is allowed
    if visitor.tokens > 0 {
        visitor.tokens--
        return true
    }

    return false
}

func RateLimiterMiddleware(rl *RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        if !rl.Allow(ip) {
            // Record rate limiter rejection in metrics
            metrics.RateLimiterRejections.WithLabelValues(ip).Inc()
            
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many requests. Please try again later.",
            })
            return
        }
        c.Next()
    }
}