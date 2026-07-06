package services

import (
	"api/internal/testutils"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCatalogFromID_Success(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("catalog-1", "Main Catalog"))

	catalog, err := GetCatalogFromID("catalog-1")

	require.NoError(t, err)
	assert.Equal(t, "Main Catalog", catalog.Name)
}

func TestGetCatalogFromID_EmptyID(t *testing.T) {
	_, err := GetCatalogFromID("")

	assert.ErrorIs(t, err, ErrCatalogNotFound)
}

func TestGetAddressFromCatalogId_Success(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", "http://bee-api.local"))

	address, err := GetAddressFromCatalogId("catalog-1")

	require.NoError(t, err)
	assert.Equal(t, "http://bee-api.local", address)
}

func TestGetAddressFromCatalogId_NotFound(t *testing.T) {
	mock := testutils.UseMockDB(t)

	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}))

	_, err := GetAddressFromCatalogId("catalog-1")

	assert.ErrorIs(t, err, ErrCatalogNotFound)
}

// TestSubmitCorrectAnswer mirrors the report's cas de test: BeeAPI reports a
// match, so CheckPuzzleAnswer must return true.
func TestSubmitCorrectAnswer(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/puzzle/check/first", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"matches": true}`))
	}))
	defer beeAPI.Close()

	mock := testutils.UseMockDB(t)
	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))

	isCorrect, err := CheckPuzzleAnswer("catalog-1", "theme-1", "puzzle-1", 1, "user-1", "42")

	require.NoError(t, err)
	assert.True(t, isCorrect)
}

// TestSubmitWrongAnswer mirrors the report's cas de test: BeeAPI reports no
// match, so CheckPuzzleAnswer must return false without erroring.
func TestSubmitWrongAnswer(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"matches": false}`))
	}))
	defer beeAPI.Close()

	mock := testutils.UseMockDB(t)
	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))

	isCorrect, err := CheckPuzzleAnswer("catalog-1", "theme-1", "puzzle-1", 1, "user-1", "wrong")

	require.NoError(t, err)
	assert.False(t, isCorrect)
}

func TestCheckPuzzleAnswer_SecondStepURL(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/puzzle/check/second", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"matches": true}`))
	}))
	defer beeAPI.Close()

	mock := testutils.UseMockDB(t)
	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))

	isCorrect, err := CheckPuzzleAnswer("catalog-1", "theme-1", "puzzle-1", 2, "user-1", "42")

	require.NoError(t, err)
	assert.True(t, isCorrect)
}

func TestCheckPuzzleAnswer_InvalidStep(t *testing.T) {
	_, err := CheckPuzzleAnswer("catalog-1", "theme-1", "puzzle-1", 99, "user-1", "42")

	assert.ErrorIs(t, err, ErrInvalidStep)
}

func TestCheckPuzzleAnswer_BeeAPIError(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer beeAPI.Close()

	mock := testutils.UseMockDB(t)
	mock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))

	_, err := CheckPuzzleAnswer("catalog-1", "theme-1", "puzzle-1", 1, "user-1", "42")

	assert.ErrorIs(t, err, ErrSolutionCheckFailed)
}

func TestGetPuzzleInput_CacheHit(t *testing.T) {
	redisMock := testutils.UseMockRedis(t)

	cacheKey := "catalog_puzzle_input:catalog-1:theme-1:puzzle-1:user-1"
	redisMock.ExpectGet(cacheKey).SetVal(`{"n": 42}`)

	input, err := GetPuzzleInput("catalog-1", "theme-1", "puzzle-1", "user-1", context.Background())

	require.NoError(t, err)
	assert.Equal(t, float64(42), input["n"])
}

func TestGetPuzzleInput_CacheMissFetchesFromBeeAPI(t *testing.T) {
	beeAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/puzzle/generate/input", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"n": 7}`))
	}))
	defer beeAPI.Close()

	redisMock := testutils.UseMockRedis(t)
	dbMock := testutils.UseMockDB(t)

	cacheKey := "catalog_puzzle_input:catalog-1:theme-1:puzzle-1:user-1"
	redisMock.ExpectGet(cacheKey).RedisNil()
	dbMock.ExpectQuery(`SELECT (.+) FROM "catalogs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow("catalog-1", beeAPI.URL))
	redisMock.ExpectSet(cacheKey, []byte(`{"n":7}`), 0).SetVal("OK")

	input, err := GetPuzzleInput("catalog-1", "theme-1", "puzzle-1", "user-1", context.Background())

	require.NoError(t, err)
	assert.Equal(t, float64(7), input["n"])
	assert.NoError(t, redisMock.ExpectationsWereMet())
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
