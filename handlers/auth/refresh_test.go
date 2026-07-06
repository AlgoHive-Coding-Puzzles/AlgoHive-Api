package auth

import (
	"api/config"
	"api/internal/testutils"
	"api/utils"
	"net/http"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefreshTokenRotation mirrors the report's cas de test: refreshing a
// valid token must return a brand new token, distinct from the one
// presented, and blacklist the old one in Redis.
func TestRefreshTokenRotation(t *testing.T) {
	withAuthJWTConfig(t)
	redisMock := testutils.UseMockRedis(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	oldToken, err := utils.GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	redisMock.Regexp().ExpectExists(`token:blacklist:.+`).SetVal(0)
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "blocked"}).AddRow("user-1", "user@example.com", false))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("token:blacklist:"+oldToken, "1", time.Hour).SetVal("OK")

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)
	testutils.SetAuthCookie(c, oldToken)

	RefreshToken(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp AuthResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEqual(t, oldToken, resp.Token)
	assert.Equal(t, "user-1", resp.UserID)
	assert.NoError(t, redisMock.ExpectationsWereMet())
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestRefreshToken_NoToken(t *testing.T) {
	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrNoTokenProvided)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	withAuthJWTConfig(t)

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)
	testutils.SetAuthCookie(c, "garbage-token")

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrInvalidToken)
}

func TestRefreshToken_ExpiredToken(t *testing.T) {
	withAuthJWTConfig(t)

	token := generateTestExpiredJWT(t, "user-1", "user@example.com")

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)
	testutils.SetAuthCookie(c, token)

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrInvalidExpiredToken)
}

func TestRefreshToken_BlacklistedToken(t *testing.T) {
	withAuthJWTConfig(t)
	redisMock := testutils.UseMockRedis(t)

	token, err := utils.GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	redisMock.Regexp().ExpectExists(`token:blacklist:.+`).SetVal(1)

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)
	testutils.SetAuthCookie(c, token)

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrInvalidToken)
}

func TestRefreshToken_BlockedAccount(t *testing.T) {
	withAuthJWTConfig(t)
	redisMock := testutils.UseMockRedis(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	token, err := utils.GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	redisMock.Regexp().ExpectExists(`token:blacklist:.+`).SetVal(0)
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "blocked"}).AddRow("user-1", "user@example.com", true))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/refresh", nil)
	testutils.SetAuthCookie(c, token)

	RefreshToken(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrAccountBlocked)
}

// generateTestExpiredJWT builds a token whose expiration is already in the past.
func generateTestExpiredJWT(t *testing.T, userID, email string) string {
	t.Helper()
	claims := &utils.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(config.JWTSecret))
	require.NoError(t, err)
	return signed
}
