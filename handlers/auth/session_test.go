package auth

import (
	"api/internal/testutils"
	"net/http"
	"testing"

	"api/utils"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogout_Success(t *testing.T) {
	withAuthJWTConfig(t)
	redisMock := testutils.UseMockRedis(t)

	token, err := utils.GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	// The blacklist TTL is derived from the token's real remaining lifetime,
	// so it can't be matched exactly; only the command shape is asserted
	// (a non-zero placeholder expiration keeps the same arg count as the
	// real PX-bearing call).
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("token:blacklist:"+token, "1", time.Hour).SetVal("OK")
	redisMock.Regexp().ExpectDel(`user_session:.+`).SetVal(1)

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/logout", nil)
	testutils.SetAuthCookie(c, token)

	Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), ErrLogoutSuccess)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestLogout_NoToken(t *testing.T) {
	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/logout", nil)

	Logout(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrNoTokenProvided)
}

func TestCheckAuth_ValidToken(t *testing.T) {
	withAuthJWTConfig(t)
	redisMock := testutils.UseMockRedis(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	token, err := utils.GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	redisMock.Regexp().ExpectExists(`token:blacklist:.+`).SetVal(0)
	redisMock.Regexp().ExpectGet(`user_session:.+`).RedisNil()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "blocked"}).AddRow("user-1", "user@example.com", false))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("user_session:user-1", "ignored", 5*time.Minute).SetVal("OK")

	c, w := testutils.NewTestContext(t, http.MethodGet, "/auth/check", nil)
	testutils.SetAuthCookie(c, token)

	CheckAuth(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":true`)
	assert.NoError(t, redisMock.ExpectationsWereMet())
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCheckAuth_NoToken(t *testing.T) {
	c, w := testutils.NewTestContext(t, http.MethodGet, "/auth/check", nil)

	CheckAuth(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":false`)
}

func TestCheckAuth_InvalidToken(t *testing.T) {
	withAuthJWTConfig(t)

	c, w := testutils.NewTestContext(t, http.MethodGet, "/auth/check", nil)
	testutils.SetAuthCookie(c, "garbage-token")

	CheckAuth(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"valid":false`)
}
