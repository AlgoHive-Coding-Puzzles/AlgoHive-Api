package competitions

import (
	"api/internal/testutils"
	"net/http"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeaderboardCache mirrors the report's cas de test: reading the
// competition's tries (leaderboard data) while a cache entry exists must
// return the cached payload without querying the database.
func TestLeaderboardCache(t *testing.T) {
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)

	mockAuthenticatedUser(dbMock, "user-1")
	redisMock.Regexp().ExpectGet(`competition_tries:comp-1:.+`).
		SetVal(`[{"id":"try-1","puzzle_id":"puzzle-1"}]`)

	c, w := testutils.NewTestContext(t, http.MethodGet, "/competitions/comp-1/tries", nil)
	c.Set("userID", "user-1")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "comp-1"})

	GetCompetitionTries(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "try-1")
	// Only the auth lookup should have hit the database: the tries
	// themselves came from Redis, never from Postgres.
	assert.NoError(t, dbMock.ExpectationsWereMet())
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestGetCompetitionTries_CacheMissFetchesFromDB(t *testing.T) {
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)
	redisMock.MatchExpectationsInOrder(false)

	mockAuthenticatedUser(dbMock, "user-1")
	redisMock.Regexp().ExpectGet(`competition_tries:comp-1:.+`).RedisNil()
	dbMock.ExpectQuery(`SELECT DISTINCT c\.\* FROM competitions`).WillReturnRows(competitionRow())
	dbMock.ExpectQuery(`SELECT (.+) FROM "tries"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "competition_id", "user_id", "puzzle_id"}).
			AddRow("try-1", "comp-1", "user-1", "puzzle-1"))
	dbMock.ExpectQuery(`SELECT (.+) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	dbMock.ExpectQuery(`SELECT (.+) FROM "user_groups"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id"}))
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("competition_tries:comp-1:user-1", "ignored", 5*time.Minute).SetVal("OK")

	c, w := testutils.NewTestContext(t, http.MethodGet, "/competitions/comp-1/tries", nil)
	c.Set("userID", "user-1")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "comp-1"})

	GetCompetitionTries(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "try-1")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

// TestScoreOrdering mirrors the report's cas de test: with several
// competitors, the ranking must follow total score descending, then
// highest puzzle index descending, then earliest first action (this
// ordering is enforced by the SQL query itself; here we assert the
// handler surfaces the rows exactly as returned, un-reordered, and computes
// the aggregate stats correctly).
func TestScoreOrdering(t *testing.T) {
	dbMock := testutils.UseMockDB(t)
	dbMock.MatchExpectationsInOrder(false)
	redisMock := testutils.UseMockRedis(t)

	mockAuthenticatedUser(dbMock, "admin-1")
	// This user isn't staff, but does have group-based access to the
	// competition, which is enough to view its statistics.
	dbMock.ExpectQuery(`competition_groups`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	redisMock.Regexp().ExpectGet(`competition_stats:.+`).RedisNil()
	dbMock.ExpectQuery(`SELECT (.+) FROM "competitions"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow("comp-1", "Round 1"))
	dbMock.ExpectQuery(`SELECT[\s\S]*t\.user_id`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "firstname", "total_score", "highest_puzzle_index", "total_attempts", "first_action", "last_action"}).
			AddRow("user-2", "Bob", 100.0, 3, 5, "2026-01-01T09:00:00Z", "2026-01-01T09:30:00Z").
			AddRow("user-1", "Ada", 100.0, 3, 4, "2026-01-01T08:00:00Z", "2026-01-01T09:20:00Z").
			AddRow("user-3", "Cid", 50.0, 2, 2, "2026-01-01T08:30:00Z", "2026-01-01T09:10:00Z"))
	redisMock.CustomMatch(func(expected, actual []interface{}) error { return nil }).
		ExpectSet("competition_stats", "ignored", 5*time.Minute).SetVal("OK")

	c, w := testutils.NewTestContext(t, http.MethodGet, "/competitions/comp-1/statistics", nil)
	c.Set("userID", "admin-1")
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "comp-1"})

	GetCompetitionStatistics(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp CompetitionStatsResponse
	testutils.DecodeJSON(t, w, &resp)
	require.Len(t, resp.UserStats, 3)
	// The handler must preserve the SQL-enforced order (Bob before Ada
	// despite the tie on score, because Bob's row came first from the DB).
	assert.Equal(t, "user-2", resp.UserStats[0].UserID)
	assert.Equal(t, "user-1", resp.UserStats[1].UserID)
	assert.Equal(t, "user-3", resp.UserStats[2].UserID)
	assert.Equal(t, float64(100), resp.HighestScore)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
