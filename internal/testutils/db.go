// Package testutils provides shared helpers for mocking AlgoHive-Api's
// external dependencies (Postgres/GORM, Redis, gin) in unit tests, so
// business logic can be exercised without a real database or cache.
package testutils

import (
	"api/database"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewMockDB creates a *gorm.DB backed by a sqlmock connection, so tests can
// assert on the SQL GORM generates and return canned rows/results.
func NewMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return gdb, mock
}

// UseMockDB points the global database.DB at a mocked connection for the
// duration of the test, and restores the previous value afterwards.
func UseMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()

	gdb, mock := NewMockDB(t)

	previous := database.DB
	database.DB = gdb
	t.Cleanup(func() {
		database.DB = previous
	})

	return mock
}
