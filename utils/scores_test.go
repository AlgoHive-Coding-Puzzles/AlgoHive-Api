package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateBaseScore(t *testing.T) {
	tests := []struct {
		lvl  string
		step int
		want int
	}{
		{"EASY", 1, 15},
		{"EASY", 2, 35},
		{"MEDIUM", 1, 35},
		{"MEDIUM", 2, 65},
		{"HARD", 1, 65},
		{"HARD", 2, 135},
		{"UNKNOWN", 1, 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, calculateBaseScore(tt.lvl, tt.step))
	}
}

func TestCalculateMultiplier(t *testing.T) {
	assert.Equal(t, float32(1), CalculateMultiplier(0))
	assert.Equal(t, float32(1.05), CalculateMultiplier(5))
	assert.Equal(t, float32(1.15), CalculateMultiplier(15))
}

func TestCalculateMalusFromAttempts(t *testing.T) {
	assert.Equal(t, 0, CalculateMalusFromAttempts(1))
	assert.Equal(t, 0, CalculateMalusFromAttempts(3))
	assert.Equal(t, -2, CalculateMalusFromAttempts(4))
	assert.Equal(t, -2, CalculateMalusFromAttempts(5))
}

// TestCalculatePointsFromTimeSpent_RFC3339 is a regression test for a bug
// found while implementing the test harness: StartTime/EndTime are stored
// using time.RFC3339 (see services.TriggerPuzzleFirstTry/EndTry), but
// calculateTimeSpent used to parse them with an incompatible custom layout,
// silently returning 0 for every submission.
func TestCalculatePointsFromTimeSpent_RFC3339(t *testing.T) {
	start := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	points := CalculatePointsFromTimeSpent("EASY", start, end)

	assert.Equal(t, 20, points, "expected the fast-solve bonus for a 5 minute EASY puzzle")
}

// TestCalculatePointsFromTimeSpent_InvalidFormat documents the current
// behavior: an unparsable timestamp falls back to a 0 minute duration, which
// for EASY puzzles happens to fall in the "< 10 minutes" bonus bracket.
func TestCalculatePointsFromTimeSpent_InvalidFormat(t *testing.T) {
	points := CalculatePointsFromTimeSpent("EASY", "not-a-date", "also-not-a-date")

	assert.Equal(t, 20, points)
}

func TestCalculateScore_EndToEnd(t *testing.T) {
	start := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	// EASY step 1 (base 15) + fast solve bonus (20) + no malus (0 attempts) = 35
	// multiplier for puzzleIndex 0 is 1.0
	score := CalculateScore("EASY", 0, 1, start, end, 1)

	assert.Equal(t, float64(35), score)
}
