package middleware

import (
	"api/config"
	"api/internal/testutils"
	"api/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestJWT builds a signed token with an arbitrary time-to-live,
// including negative durations to produce already-expired tokens.
func generateTestJWT(t *testing.T, userID, email string, ttl time.Duration) string {
	t.Helper()
	claims := &utils.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(config.JWTSecret))
	require.NoError(t, err)
	return signed
}

func withAuthConfig(t *testing.T, secret string, expirationSeconds int) {
	t.Helper()
	prevSecret, prevExp := config.JWTSecret, config.JWTExpiration
	config.JWTSecret = secret
	config.JWTExpiration = expirationSeconds
	t.Cleanup(func() {
		config.JWTSecret = prevSecret
		config.JWTExpiration = prevExp
	})
}

func newAuthTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func TestGetTokenFromRequest_Cookie(t *testing.T) {
	c, _ := newAuthTestContext(http.MethodGet, "/")
	c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token"})

	token, err := getTokenFromRequest(c)

	require.NoError(t, err)
	assert.Equal(t, "cookie-token", token)
}

func TestGetTokenFromRequest_BearerHeader(t *testing.T) {
	c, _ := newAuthTestContext(http.MethodGet, "/")
	c.Request.Header.Set("Authorization", "Bearer header-token")

	token, err := getTokenFromRequest(c)

	require.NoError(t, err)
	assert.Equal(t, "header-token", token)
}

func TestGetTokenFromRequest_Missing(t *testing.T) {
	c, _ := newAuthTestContext(http.MethodGet, "/")

	_, err := getTokenFromRequest(c)

	assert.Error(t, err)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)
	mock := testutils.UseMockRedis(t)

	// Generate a real token via the auth package's own signing helper by
	// building it directly with the same secret/claims shape.
	token := generateTestJWT(t, "user-1", "user@example.com", time.Hour)

	mock.ExpectExists("token:blacklist:" + token).SetVal(0)

	router := gin.New()
	router.GET("/secure", AuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-1")
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	router := gin.New()
	router.GET("/secure", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)

	router := gin.New()
	router.GET("/secure", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "garbage-token"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)

	token := generateTestJWT(t, "user-1", "user@example.com", -time.Hour)

	router := gin.New()
	router.GET("/secure", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_BlacklistedToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)
	mock := testutils.UseMockRedis(t)

	token := generateTestJWT(t, "user-1", "user@example.com", time.Hour)
	mock.ExpectExists("token:blacklist:" + token).SetVal(1)

	router := gin.New()
	router.GET("/secure", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalAuthMiddleware_NoToken(t *testing.T) {
	router := gin.New()
	router.GET("/opt", OptionalAuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":""`)
}

func TestAddKeyMiddleware_InvalidTokenContinues(t *testing.T) {
	router := gin.New()
	router.GET("/opt", AddKeyMiddleware(), func(c *gin.Context) {
		_, exists := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"has_user": exists})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "garbage"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "false")
}

func TestAddKeyMiddleware_ValidToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)
	token := generateTestJWT(t, "user-1", "user@example.com", time.Hour)

	router := gin.New()
	router.GET("/opt", AddKeyMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-1")
}

func TestOptionalAuthMiddleware_ValidToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)
	mock := testutils.UseMockRedis(t)

	token := generateTestJWT(t, "user-1", "user@example.com", time.Hour)
	mock.ExpectExists("token:blacklist:" + token).SetVal(0)

	router := gin.New()
	router.GET("/opt", OptionalAuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-1")
}

func TestOptionalAuthMiddleware_BlacklistedToken(t *testing.T) {
	withAuthConfig(t, "test-secret", 3600)
	mock := testutils.UseMockRedis(t)

	token := generateTestJWT(t, "user-1", "user@example.com", time.Hour)
	mock.ExpectExists("token:blacklist:" + token).SetVal(1)

	router := gin.New()
	router.GET("/opt", OptionalAuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get(ContextKeyUserID)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":""`)
}
