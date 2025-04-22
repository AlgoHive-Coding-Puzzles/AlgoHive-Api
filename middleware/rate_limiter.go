package middleware

import (
	"api/metrics"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LimitMode defines different methods to identify users for rate limiting
type LimitMode int

const (
	IPOnly LimitMode = iota      // Use only IP address (default)
	IPAndSession                 // Use IP + session ID (better for LAN environments)
	UserBased                    // Use user ID (requires authentication)
)

type RateLimiter struct {
    visitors map[string]*Visitor
    mu       sync.Mutex
    rate     int           // Maximum requests per minute
    burst    int           // Burst capacity
    interval time.Duration // Refill interval
    mode     LimitMode     // Identification mode for rate limiting
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
        mode:     IPOnly, // Default to IP-only mode
    }
}

// SetMode changes the identifier mode used by the rate limiter
func (rl *RateLimiter) SetMode(mode LimitMode) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    rl.mode = mode
    // Clear existing visitors when changing mode
    rl.visitors = make(map[string]*Visitor)
}

// getIdentifier returns the appropriate identifier for the current request based on mode
func getIdentifier(c *gin.Context, mode LimitMode) string {
    ip := c.ClientIP()
    
    switch mode {
    case IPAndSession:
        // Use IP + session ID if available
        sessionID, _ := c.Cookie("session_id")
        if sessionID != "" {
            return ip + ":" + sessionID
        }
        // Fallback to IP + User-Agent if session not available
        userAgent := c.Request.UserAgent()
        return ip + ":" + userAgent
        
    case UserBased:
        // Use user ID if authenticated
        if userID, exists := c.Get("user_id"); exists {
            return userID.(string)
        }
        // Fallback to IP if not authenticated
        return ip
        
    default: // IPOnly
        return ip
    }
}

func (rl *RateLimiter) getVisitor(identifier string) *Visitor {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    if visitor, exists := rl.visitors[identifier]; exists {
        return visitor
    }

    visitor := &Visitor{
        tokens:      rl.burst,
        lastUpdated: time.Now(),
    }
    rl.visitors[identifier] = visitor
    return visitor
}

func (rl *RateLimiter) Allow(identifier string) bool {
    visitor := rl.getVisitor(identifier)

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
        identifier := getIdentifier(c, rl.mode)
        if !rl.Allow(identifier) {
            // Record rate limiter rejection in metrics
            metrics.RateLimiterRejections.WithLabelValues(identifier).Inc()
            
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many requests. Please try again later.",
            })
            return
        }
        c.Next()
    }
}

// EnableLANMode configures the rate limiter to work better in LAN environments
func (rl *RateLimiter) EnableLANMode() {
    rl.SetMode(IPAndSession)
}