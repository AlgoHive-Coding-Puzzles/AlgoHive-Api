package services

import (
	"api/internal/testutils"
	"api/models"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerPuzzleFirstTry_ReturnsExistingTry(t *testing.T) {
	mock := testutils.UseMockDB(t)
	mock.MatchExpectationsInOrder(true)

	competition := models.Competition{ID: "comp-1"}
	user := models.User{ID: "user-1"}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "step", "attempts", "score"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, 1, 0, 0))
	mock.ExpectCommit()

	try, err := TriggerPuzzleFirstTry(competition, "puzzle-1", 0, "EASY", user)

	require.NoError(t, err)
	assert.Equal(t, "try-1", try.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTriggerPuzzleFirstTry_CreatesNewTry(t *testing.T) {
	mock := testutils.UseMockDB(t)
	mock.MatchExpectationsInOrder(false)

	competition := models.Competition{ID: "comp-1"}
	user := models.User{ID: "user-1"}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`INSERT INTO "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("try-new"))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	try, err := TriggerPuzzleFirstTry(competition, "puzzle-1", 0, "EASY", user)

	require.NoError(t, err)
	assert.Equal(t, "try-new", try.ID)
	assert.Equal(t, "puzzle-1", try.PuzzleID)
	assert.Equal(t, 1, try.Step)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPuzzleTry_NotFound(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := GetPuzzleTry("comp-1", "puzzle-1", 0, 1, "user-1")

	assert.Error(t, err)
}

func TestUpdateTry_AlreadyFinished(t *testing.T) {
	mock := testutils.UseMockDB(t)

	competition := models.Competition{ID: "comp-1"}
	user := models.User{ID: "user-1"}
	endTime := "2026-01-01T10:00:00Z"

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "step", "attempts", "score", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, 1, 1, 15, endTime))
	mock.ExpectRollback()

	_, err := UpdateTry(competition, "puzzle-1", 0, 1, user, "wrong answer")

	assert.ErrorIs(t, err, ErrTryAlreadyFinished)
}

func TestUpdateTry_IncrementsAttempts(t *testing.T) {
	mock := testutils.UseMockDB(t)
	mock.MatchExpectationsInOrder(false)

	competition := models.Competition{ID: "comp-1"}
	user := models.User{ID: "user-1"}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "step", "attempts", "score", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, 1, 1, 0, nil))
	mock.ExpectExec(`UPDATE "tries"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	updated, err := UpdateTry(competition, "puzzle-1", 0, 1, user, "wrong answer")

	require.NoError(t, err)
	assert.Equal(t, 2, updated.Attempts)
	assert.Equal(t, "wrong answer", *updated.LastAnswer)
}

func TestEndTry_CalculatesScoreAndCreatesNextStep(t *testing.T) {
	mock := testutils.UseMockDB(t)
	mock.MatchExpectationsInOrder(false)

	competition := models.Competition{ID: "comp-1"}
	user := models.User{ID: "user-1"}
	start := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 0, 0, start, nil))
	mock.ExpectExec(`UPDATE "tries"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("try-step2"))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	finished, err := EndTry(competition, "puzzle-1", 0, 1, user, "correct answer")

	require.NoError(t, err)
	assert.Equal(t, 1, finished.Attempts)
	assert.Greater(t, finished.Score, float64(0))
	assert.NotNil(t, finished.EndTime)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPuzzleFirstTry_DelegatesToStepOne(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "step"}).AddRow("try-1", 1))

	try, err := GetPuzzleFirstTry("comp-1", "puzzle-1", 0, "user-1")

	require.NoError(t, err)
	assert.Equal(t, "try-1", try.ID)
}

func TestGetPuzzleTries_Success(t *testing.T) {
	mock := testutils.UseMockDB(t)
	competition := models.Competition{ID: "comp-1"}

	mock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "step"}).
			AddRow("try-1", 1).
			AddRow("try-2", 2))

	tries, err := GetPuzzleTries(competition, "puzzle-1", 0, "user-1")

	require.NoError(t, err)
	assert.Len(t, tries, 2)
}

func TestUserHasPermissionToViewPuzzle_FirstPuzzleAlwaysAllowed(t *testing.T) {
	competition := models.Competition{ID: "comp-1"}

	assert.True(t, UserHasPermissionToViewPuzzle(competition, 0, "user-1"))
}

func TestUserHasPermissionToViewPuzzle_RequiresPreviousSolved(t *testing.T) {
	mock := testutils.UseMockDB(t)
	competition := models.Competition{ID: "comp-1"}

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	assert.True(t, UserHasPermissionToViewPuzzle(competition, 1, "user-1"))
}
