package services

import (
	"api/internal/testutils"
	"api/models"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAccessibleCompetition_Granted(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "show"}).AddRow("comp-1", "Round 1", true))

	var competition models.Competition
	err := GetAccessibleCompetition("user-1", "comp-1", &competition)

	require.NoError(t, err)
	assert.Equal(t, "comp-1", competition.ID)
}

func TestGetAccessibleCompetition_MissingParams(t *testing.T) {
	var competition models.Competition
	err := GetAccessibleCompetition("", "comp-1", &competition)

	assert.ErrorIs(t, err, ErrCompetitionAccessDenied)
}

func TestGetAccessibleCompetition_Denied(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "show"}))

	var competition models.Competition
	err := GetAccessibleCompetition("user-1", "comp-1", &competition)

	assert.ErrorIs(t, err, ErrCompetitionAccessDenied)
}

func TestCompetitionExists(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	assert.True(t, CompetitionExists("comp-1"))
}

func TestCompetitionExists_EmptyID(t *testing.T) {
	assert.False(t, CompetitionExists(""))
}

func TestGetCompetitionByID_NotFound(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "competitions"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := GetCompetitionByID("comp-1")

	assert.ErrorIs(t, err, ErrCompetitionNotFound)
}

func TestGetUserCompetitions_MissingUserID(t *testing.T) {
	_, err := GetUserCompetitions("")

	assert.Error(t, err)
}

func TestGetUserCompetitions_Success(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "show"}).
			AddRow("comp-1", "Round 1", true).
			AddRow("comp-2", "Round 2", true))

	competitions, err := GetUserCompetitions("user-1")

	require.NoError(t, err)
	assert.Len(t, competitions, 2)
}
