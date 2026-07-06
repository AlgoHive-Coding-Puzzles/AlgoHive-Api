package competitions

import (
	"api/internal/testutils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthenticatedUser configures the sqlmock expectations for
// middleware.GetUserFromRequest, which every handler in this package calls
// first: a users row lookup with a Roles preload.
func mockAuthenticatedUser(dbMock sqlmock.Sqlmock, userID string) {
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_roles"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role_id"}))
}

func competitionRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "title", "catalog_id", "catalog_theme", "show"}).
		AddRow("comp-1", "Round 1", "catalog-1", "theme-1", true)
}

// TestSubmitCorrectAnswer_Handler mirrors the report's cas de test at the
// handler level: a correct submission for step 1 must update the score,
// mark the try finished, and report is_correct=true.
func TestSubmitCorrectAnswer_Handler(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"matches": true}`))
	}))
	defer beeAPI.Close()

	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)

	mockAuthenticatedUser(dbMock, "user-1")
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 0, 0, "2026-01-01T10:00:00Z", nil))
	dbMock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 0, 0, "2026-01-01T10:00:00Z", nil))
	dbMock.ExpectExec(`UPDATE "tries"`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectQuery(`INSERT INTO "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("try-2"))
	dbMock.ExpectCommit()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))
	redisMock.Regexp().ExpectDel(`comp_puzzle_input:.+`).SetVal(1)

	c, w := testutils.NewTestContext(t, http.MethodPost, "/competitions/answer_puzzle", CompetitionTry{
		CompetitionID: "comp-1",
		PuzzleId:      "puzzle-1",
		PuzzleIndex:   0,
		PuzzleStep:    1,
		Answer:        "42",
	})
	c.Set("userID", "user-1")

	AnswerPuzzle(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PuzzleAnswerResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.True(t, resp.IsCorrect)
	assert.False(t, resp.AlreadySolved)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

// TestSubmitWrongAnswer_Handler mirrors the report's cas de test: an
// incorrect submission must leave the try unfinished and report
// is_correct=false, with the attempt counter incremented.
func TestSubmitWrongAnswer_Handler(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"matches": false}`))
	}))
	defer beeAPI.Close()

	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	mockAuthenticatedUser(dbMock, "user-1")
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 0, 0, "2026-01-01T10:00:00Z", nil))
	dbMock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 0, 0, "2026-01-01T10:00:00Z", nil))
	dbMock.ExpectExec(`UPDATE "tries"`).WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/competitions/answer_puzzle", CompetitionTry{
		CompetitionID: "comp-1",
		PuzzleId:      "puzzle-1",
		PuzzleIndex:   0,
		PuzzleStep:    1,
		Answer:        "wrong",
	})
	c.Set("userID", "user-1")

	AnswerPuzzle(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PuzzleAnswerResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.False(t, resp.IsCorrect)
	assert.False(t, resp.AlreadySolved)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// TestSubmitDuplicateCorrect mirrors the report's cas de test: resubmitting
// for a puzzle step that was already solved must not touch the score or
// call BeeAPI again, and must report already_solved=true.
func TestSubmitDuplicateCorrect(t *testing.T) {
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)

	mockAuthenticatedUser(dbMock, "user-1")
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "puzzle_lvl", "step", "attempts", "score", "start_time", "end_time"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, "EASY", 1, 1, 35, "2026-01-01T10:00:00Z", "2026-01-01T10:05:00Z"))

	c, w := testutils.NewTestContext(t, http.MethodPost, "/competitions/answer_puzzle", CompetitionTry{
		CompetitionID: "comp-1",
		PuzzleId:      "puzzle-1",
		PuzzleIndex:   0,
		PuzzleStep:    1,
		Answer:        "42",
	})
	c.Set("userID", "user-1")

	AnswerPuzzle(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PuzzleAnswerResponse
	testutils.DecodeJSON(t, w, &resp)
	assert.True(t, resp.IsCorrect)
	assert.True(t, resp.AlreadySolved)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGetInputFromCompetition_CacheHit(t *testing.T) {
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)

	mockAuthenticatedUser(dbMock, "user-1")
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	// TriggerPuzzleFirstTry finds an existing try for this puzzle/step.
	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "step"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, 1))
	dbMock.ExpectCommit()
	redisMock.Regexp().ExpectGet(`comp_puzzle_input:.+`).SetVal(`{"n":1}`)

	c, w := testutils.NewTestContext(t, http.MethodPost, "/competitions/input", InputRequest{
		CompetitionID: "comp-1",
		PuzzleID:      "puzzle-1",
		PuzzleIndex:   0,
	})
	c.Set("userID", "user-1")

	GetInputFromCompetition(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestGetInputFromCompetition_CacheMiss(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/puzzle/generate/input", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"n": 5}`))
	}))
	defer beeAPI.Close()

	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)
	redisMock.MatchExpectationsInOrder(false)

	mockAuthenticatedUser(dbMock, "user-1")
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	dbMock.ExpectBegin()
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "competition_id", "puzzle_id", "puzzle_index", "step"}).
			AddRow("try-1", "user-1", "comp-1", "puzzle-1", 0, 1))
	dbMock.ExpectCommit()
	// The handler's own cache lookup (comp_puzzle_input:*) misses...
	redisMock.Regexp().ExpectGet(`comp_puzzle_input:.+`).RedisNil()
	dbMock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))
	// ...so services.GetPuzzleInput runs its own (separately keyed) cache
	// lookup and best-effort write, in addition to the handler's own write.
	redisMock.Regexp().ExpectGet(`catalog_puzzle_input:.+`).RedisNil()
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("catalog_puzzle_input", "ignored-value", 0).SetVal("OK")
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("comp_puzzle_input", "ignored-value", 30*time.Minute).SetVal("OK")

	c, w := testutils.NewTestContext(t, http.MethodPost, "/competitions/input", InputRequest{
		CompetitionID: "comp-1",
		PuzzleID:      "puzzle-1",
		PuzzleIndex:   0,
	})
	c.Set("userID", "user-1")

	GetInputFromCompetition(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"n":5`)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
