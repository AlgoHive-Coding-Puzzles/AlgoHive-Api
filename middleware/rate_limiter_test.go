package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimiter_AllowsWithinBurst(t *testing.T) {
	rl := NewRateLimiter(10, 3) // 10/min, burst of 3

	assert.True(t, rl.Allow("client-1"))
	assert.True(t, rl.Allow("client-1"))
	assert.True(t, rl.Allow("client-1"))
}

// TestRateLimiter_BlocksBeyondBurst mirrors the report's TestLoginRateLimit:
// once the burst capacity is exhausted, further requests within the same
// window must be rejected.
func TestRateLimiter_BlocksBeyondBurst(t *testing.T) {
	rl := NewRateLimiter(10, 3)

	for i := 0; i < 3; i++ {
		require.True(t, rl.Allow("client-1"))
	}

	assert.False(t, rl.Allow("client-1"))
}

func TestRateLimiter_IndependentPerIdentifier(t *testing.T) {
	rl := NewRateLimiter(10, 1)

	assert.True(t, rl.Allow("client-1"))
	assert.False(t, rl.Allow("client-1"))
	// A different identifier has its own bucket.
	assert.True(t, rl.Allow("client-2"))
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(60, 1) // 60/min => refills 1 token/sec
	rl.interval = 50 * time.Millisecond

	require.True(t, rl.Allow("client-1"))
	require.False(t, rl.Allow("client-1"))

	time.Sleep(60 * time.Millisecond)

	assert.True(t, rl.Allow("client-1"))
}

func TestRateLimiter_RetryAfterSeconds(t *testing.T) {
	rl := NewRateLimiter(60, 20) // 1 token/sec on average
	assert.Equal(t, 1, rl.RetryAfterSeconds())

	rl2 := NewRateLimiter(1, 5) // 1 token/min
	assert.Equal(t, 60, rl2.RetryAfterSeconds())
}

func TestRateLimiterMiddleware_TooManyRequestsWithRetryAfter(t *testing.T) {
	rl := NewRateLimiter(10, 1)

	router := gin.New()
	router.GET("/ping", RateLimiterMiddleware(rl), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First request consumes the single burst token.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request from the same client must be rejected.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.NotEmpty(t, w2.Header().Get("Retry-After"))
}

func TestRateLimiter_SetMode(t *testing.T) {
	rl := NewRateLimiter(10, 1)
	require.True(t, rl.Allow("client-1"))

	rl.SetMode(UserBased)

	assert.Equal(t, UserBased, rl.mode)
	// Changing mode clears existing visitor buckets.
	assert.True(t, rl.Allow("client-1"))
}

func TestRateLimiter_EnableLANMode(t *testing.T) {
	rl := NewRateLimiter(10, 1)

	rl.EnableLANMode()

	assert.Equal(t, IPAndSession, rl.mode)
}

func TestGetIdentifier_Modes(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "203.0.113.5:1234"

	assert.Equal(t, "203.0.113.5", getIdentifier(c, IPOnly))

	c.Set("user_id", "user-42")
	assert.Equal(t, "user-42", getIdentifier(c, UserBased))

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c2.Request.RemoteAddr = "203.0.113.5:1234"
	// No user_id set: UserBased falls back to IP.
	assert.Equal(t, "203.0.113.5", getIdentifier(c2, UserBased))
}
