package services

import (
	"api/config"
	"api/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckRateLimit_NoPreviousMove(t *testing.T) {
	try := models.Try{Attempts: 10}

	limited, remaining := CheckRateLimit(try, config.DefaultRateLimitConfig)

	assert.False(t, limited)
	assert.Zero(t, remaining)
}

func TestCheckRateLimit_UnderThreshold(t *testing.T) {
	last := time.Now().Format(time.RFC3339)
	try := models.Try{Attempts: 1, LastMoveTime: &last}

	limited, _ := CheckRateLimit(try, config.DefaultRateLimitConfig)

	assert.False(t, limited)
}

// TestCheckRateLimit_FirstThreshold mirrors a wrong-answer streak (>3
// attempts) triggering the first cooldown window.
func TestCheckRateLimit_FirstThreshold(t *testing.T) {
	last := time.Now().Format(time.RFC3339)
	try := models.Try{Attempts: config.DefaultRateLimitConfig.AttemptsThreshold1, LastMoveTime: &last}

	limited, remaining := CheckRateLimit(try, config.DefaultRateLimitConfig)

	assert.True(t, limited)
	assert.Greater(t, remaining, time.Duration(0))
	assert.LessOrEqual(t, remaining, config.DefaultRateLimitConfig.CooldownDuration1)
}

func TestCheckRateLimit_SecondThresholdTakesPrecedence(t *testing.T) {
	last := time.Now().Format(time.RFC3339)
	try := models.Try{Attempts: config.DefaultRateLimitConfig.AttemptsThreshold2, LastMoveTime: &last}

	limited, remaining := CheckRateLimit(try, config.DefaultRateLimitConfig)

	assert.True(t, limited)
	assert.LessOrEqual(t, remaining, config.DefaultRateLimitConfig.CooldownDuration2)
}

func TestCheckRateLimit_CooldownExpired(t *testing.T) {
	last := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	try := models.Try{Attempts: config.DefaultRateLimitConfig.AttemptsThreshold1, LastMoveTime: &last}

	limited, remaining := CheckRateLimit(try, config.DefaultRateLimitConfig)

	assert.False(t, limited)
	assert.Zero(t, remaining)
}
