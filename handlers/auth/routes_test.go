package auth

import (
	"api/config"
	"api/internal/testutils"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestLoginRateLimit mirrors the report's cas de test: repeated failed
// login attempts from the same client must eventually be rejected with
// HTTP 429 and a Retry-After header, protecting the endpoint regardless of
// what the database layer would have returned.
func TestLoginRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevLAN := config.LANMode
	config.LANMode = false
	t.Cleanup(func() { config.LANMode = prevLAN })

	dbMock := testutils.UseMockDB(t)
	// The login rate limiter's burst is 20 (see routes.go): that many
	// requests reach the Login handler and hit the database before the
	// limiter starts rejecting requests outright.
	for i := 0; i < 20; i++ {
		dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}

	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"))

	body, _ := json.Marshal(LoginRequest{Email: "nobody@example.com", Password: "whatever1"})

	var lastCode int
	var lastRetryAfter string
	// Exceed the burst capacity to trigger a 429.
	for i := 0; i < 25; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.7:12345"
		router.ServeHTTP(w, req)
		lastCode = w.Code
		lastRetryAfter = w.Header().Get("Retry-After")
	}

	assert.Equal(t, http.StatusTooManyRequests, lastCode)
	assert.NotEmpty(t, lastRetryAfter)
}
