package testutils

import (
	"api/database"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

// NewMockRedis creates a redis.Client backed by redismock, so tests can set
// expectations on the exact Redis commands issued by the code under test.
func NewMockRedis(t *testing.T) (*redis.Client, redismock.ClientMock) {
	t.Helper()
	client, mock := redismock.NewClientMock()
	return client, mock
}

// UseMockRedis points the global database.REDIS at a mocked client for the
// duration of the test, and restores the previous value afterwards.
func UseMockRedis(t *testing.T) redismock.ClientMock {
	t.Helper()

	client, mock := NewMockRedis(t)

	previous := database.REDIS
	database.REDIS = client
	t.Cleanup(func() {
		database.REDIS = previous
	})

	return mock
}
