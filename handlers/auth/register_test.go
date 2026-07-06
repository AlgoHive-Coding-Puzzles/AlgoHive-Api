package auth

import (
	"api/internal/testutils"
	"net/http"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestRegisterUser_Success(t *testing.T) {
	withAuthJWTConfig(t)
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // no existing user with this email
	dbMock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("new-user-1"))
	dbMock.ExpectCommit()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "firstname", "lastname"}).
			AddRow("new-user-1", "grace@example.com", "Grace", "Hopper"))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/register", RegisterRequest{
		Email:     "grace@example.com",
		Firstname: "Grace",
		Lastname:  "Hopper",
		Password:  "CompilerPass1",
	})

	RegisterUser(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp AuthResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, "grace@example.com", resp.Email)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestRegisterUser_EmailInUse(t *testing.T) {
	dbMock := testutils.UseMockDB(t)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow("existing-user", "grace@example.com"))
	dbMock.ExpectRollback()

	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/register", RegisterRequest{
		Email:     "grace@example.com",
		Firstname: "Grace",
		Lastname:  "Hopper",
		Password:  "CompilerPass1",
	})

	RegisterUser(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), ErrEmailInUse)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestRegisterUser_InvalidPassword(t *testing.T) {
	c, w := testutils.NewTestContext(t, http.MethodPost, "/auth/register", RegisterRequest{
		Email:     "grace@example.com",
		Firstname: "Grace",
		Lastname:  "Hopper",
		Password:  "short",
	})

	RegisterUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
