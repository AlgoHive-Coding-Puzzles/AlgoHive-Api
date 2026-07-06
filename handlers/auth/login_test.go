package auth

import (
	"api/config"
	"api/internal/testutils"
	"api/utils"
	"net/http"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withAuthJWTConfig(t *testing.T) {
	t.Helper()
	prevSecret, prevExp := config.JWTSecret, config.JWTExpiration
	config.JWTSecret = "test-secret"
	config.JWTExpiration = 3600
	t.Cleanup(func() {
		config.JWTSecret = prevSecret
		config.JWTExpiration = prevExp
	})
}

func userRow(id, email, hashedPassword string, blocked bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "email", "password", "blocked", "firstname", "lastname"}).
		AddRow(id, email, hashedPassword, blocked, "Ada", "Lovelace")
}

// TestLoginSuccess mirrors the report's cas de test: valid credentials must
// return a JWT and HTTP 200.
func TestLoginSuccess(t *testing.T) {
	withAuthJWTConfig(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	hashed, err := utils.HashPassword("CorrectHorse1")
	require.NoError(t, err)

	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(userRow("user-1", "ada@example.com", hashed, false))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))
	// Save() on a user with (even empty) preloaded many2many associations
	// goes through GORM's association-save codepath, which wraps the
	// update in a transaction.
	dbMock.ExpectBegin()
	dbMock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "ada@example.com",
		Password: "CorrectHorse1",
	})

	Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp AuthResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "ada@example.com", resp.Email)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// TestLoginInvalidPassword mirrors the report's cas de test: a wrong
// password must return HTTP 401 with a non-technical error message and no
// token.
func TestLoginInvalidPassword(t *testing.T) {
	withAuthJWTConfig(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	hashed, err := utils.HashPassword("CorrectHorse1")
	require.NoError(t, err)

	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(userRow("user-1", "ada@example.com", hashed, false))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "ada@example.com",
		Password: "WrongPassword1",
	})

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrInvalidCredentials)
	assert.NotContains(t, w.Body.String(), "token")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestLogin_UserNotFound(t *testing.T) {
	withAuthJWTConfig(t)
	dbMock := testutils.UseMockDB(t)

	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "missing@example.com",
		Password: "whatever1",
	})

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrInvalidCredentials)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestLogin_AccountBlocked(t *testing.T) {
	withAuthJWTConfig(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	hashed, err := utils.HashPassword("CorrectHorse1")
	require.NoError(t, err)

	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(userRow("user-1", "ada@example.com", hashed, true))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "ada@example.com",
		Password: "CorrectHorse1",
	})

	Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), ErrAccountBlocked)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestLogin_EmptyCredentials(t *testing.T) {
	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/login", LoginRequest{
		Email:    "  ",
		Password: "  ",
	})

	Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
