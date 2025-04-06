package utils

import (
	"time"
)

func CalculateScore(puzzleLvl string, puzzleIndex int, step int, startTime string, endTime string, attempts int) float64 {
	baseScore := calculateBaseScore(puzzleLvl, puzzleIndex)
	scoreFromTime := CalculatePointsFromTimeSpent(puzzleLvl, startTime, endTime)
	multiplier := CalculateMultiplier(puzzleIndex)
	malus := CalculateMalusFromAttempts(attempts)

	return float64(baseScore + scoreFromTime + malus) * float64(multiplier)
}

func calculateBaseScore(puzzleLvl string, step int) int {
	switch puzzleLvl {
	case "EASY":
		switch step {
		case 1:
			return 15
		case 2:
			return 35
		}
	case "MEDIUM":
		switch step {
		case 1:
			return 35
		case 2:
			return 65
		}
	case "HARD":
		switch step {
		case 1:
			return 65
		case 2:
			return 135	
		}
	}

	return 0
}

// Return 100 + N percent of the base score
func CalculateMultiplier(puzzleIndex int) int {
	return 1 + (puzzleIndex / 100)
}

func CalculateMalusFromAttempts(attemps int) int {
	switch {
	case attemps > 3:
		return -2
	case attemps > 5:
		return -5
	case attemps > 10:
		return -10
	}
	return 0
}

func CalculatePointsFromTimeSpent(puzzleLvl string, startTime string, endTime string) int {
	// Calculate the time spent in minutes
	timeSpent := calculateTimeSpent(startTime, endTime)
	
	// Calculate points based on time spent
	switch puzzleLvl {
	case "EASY":
		switch {
			case timeSpent < 10:
				return 20
			case timeSpent < 30:
				return 10
			case timeSpent > 30:
				return -5
		}
	case "MEDIUM":
		switch {
			case timeSpent < 20:
				return 20
			case timeSpent < 40:
				return 10
			case timeSpent > 40:
				return -5
		}
	case "HARD":
		switch {
			case timeSpent < 40:
				return 75
			case timeSpent < 90:
				return 40
			case timeSpent < 120:
				return 20
		}
	}
		return 0
}

// Return a integer representing the time spent in minutes
func calculateTimeSpent(startTime string, endTime string) int {
	// Parse the time strings with the format "2006-01-02 15:04:05.000000"
	layout := "2006-01-02 15:04:05.000000"
	
	start, err := time.Parse(layout, startTime)
	if (err != nil) {
		// If there's an error parsing the time, return 0
		return 0
	}
	
	end, err := time.Parse(layout, endTime)
	if (err != nil) {
		// If there's an error parsing the time, return 0
		return 0
	}
	
	// Calculate duration between end and start times
	duration := end.Sub(start)
	
	// Convert duration to minutes and return as integer
	return int(duration.Minutes())
}