package services

import (
	"api/config"
	"api/database"
	"api/metrics"
	"api/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func TriggerPuzzleFirstTry(competition models.Competition, puzzleID string, puzzleIndex int, puzzleLvl string, userID string) (models.Try, error) {
	// Use a transaction to ensure atomicity and prevent race conditions
	tx := database.DB.Begin()
	if tx.Error != nil {
		return models.Try{}, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	
	// Defer rollback in case of error - will be ignored if successfully committed
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if a try already exists with FOR UPDATE locking to prevent concurrent operations
	var existingTry models.Try
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
			competition.ID, userID, puzzleID, 1, puzzleIndex).
		First(&existingTry).Error

	if err == nil {
		// Try already exists, commit transaction and return it
		if err := tx.Commit().Error; err != nil {
			return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
		}
		return existingTry, nil
	} else if err != gorm.ErrRecordNotFound {
		// An actual database error occurred (not just "record not found")
		tx.Rollback()
		return models.Try{}, fmt.Errorf("database error: %w", err)
	}

	// No try exists, create a new one
	newTry := models.Try{
		PuzzleID:      puzzleID,
		PuzzleIndex:   puzzleIndex,
		PuzzleLvl:     puzzleLvl,
		Step:          1,
		StartTime:     time.Now().Format(time.RFC3339),
		EndTime:       nil,
		Attempts:      0,
		LastAnswer:    nil,
		LastMoveTime:  nil,
		Score:         0,
		CompetitionID: competition.ID,
		UserID:        userID,
	}

	if err := tx.Create(&newTry).Error; err != nil {
		tx.Rollback()
		if config.Env == "development" {
			return models.Try{}, nil
		}
		return models.Try{}, fmt.Errorf("failed to create new try: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newTry, nil
}

func GetPuzzleFirstTry(competitionID string, puzzleID string, puzzleIndex int, userID string) (models.Try, error) {
	// Step 1: Check if a try already exists
	var existingTry models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
		competitionID, userID, puzzleID, 1, puzzleIndex).First(&existingTry).Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to fetch existing try: %w", err)
	}

	return existingTry, nil
}

func UpdateTry(competition models.Competition, puzzleID string, puzzleIndex int, step int, userID string, answer string) (models.Try, error) {
	// Step 1: Check if a try already exists
	var existingTry models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
		competition.ID, userID, puzzleID, step, puzzleIndex).First(&existingTry).Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to fetch existing try: %w", err)
	}

	// Step 2: Update the existing try
	moveTime := time.Now().Format(time.RFC3339)
	existingTry.LastMoveTime = &moveTime
	existingTry.LastAnswer = &answer
	existingTry.Attempts++
	if existingTry.EndTime != nil {
		return models.Try{}, fmt.Errorf("try already finished")
	}
	if err := database.DB.Save(&existingTry).Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to update existing try: %w", err)
	}

	return existingTry, nil
}

func EndTry(competition models.Competition, puzzleID string, puzzleIndex int, step int, userID string, answer string) (models.Try, error) {
	// Step 1: Check if a try already exists
	var existingTry models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
		competition.ID, userID, puzzleID, step, puzzleIndex).First(&existingTry).Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to fetch existing try: %w", err)
	}

	// Step 2: Update the existing try
	moveTime := time.Now().Format(time.RFC3339)
	existingTry.LastMoveTime = &moveTime
	existingTry.LastAnswer = &answer
	existingTry.Attempts++
	existingTry.Score = 100 // Assuming score is set to 100 when ending the try
	if existingTry.EndTime != nil {
		return models.Try{}, fmt.Errorf("try already finished")
	}
	now := time.Now().Format(time.RFC3339)
	existingTry.EndTime = &now
	if err := database.DB.Save(&existingTry).Error; err != nil {
		return models.Try{}, fmt.Errorf("failed to update existing try: %w", err)
	}

	// Step 3: Create a new try for the next step if it's the first step
	if step == 1 {
		newTry := models.Try{
			PuzzleID:      puzzleID,
			PuzzleIndex:   existingTry.PuzzleIndex,
			PuzzleLvl:     existingTry.PuzzleLvl,
			Step:          2,
			StartTime:     time.Now().Format(time.RFC3339),
			EndTime:       nil,
			Attempts:      0,
			LastAnswer:    nil,
			LastMoveTime:  nil,
			Score:         0,
			CompetitionID: competition.ID,
			UserID:        userID,
		}
		if err := database.DB.Create(&newTry).Error; err != nil {
			return models.Try{}, fmt.Errorf("failed to create new try: %w", err)
		}
	}

	return existingTry, nil
}

func GetPuzzleTries(competition models.Competition, puzzleID string, puzzleIndex int, userID string) ([]models.Try, error) {
	var tries []models.Try
	err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND puzzle_index = ?",
		competition.ID, userID, puzzleID, puzzleIndex).Find(&tries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tries: %w", err)
	}
	return tries, nil
}

// The User has permission to view the puzzle if:
// 1. The puzzleIndex is 0
// 2. Or if the puzzle[puzzleIndex-1] is already solved (puzzle solved means that the two tries have ent_time != nil)
func UserHasPermissionToViewPuzzle(competition models.Competition, puzzleIndex int, UserID string) bool {
	if puzzleIndex == 0 {
		return true
	}

	var tries []models.Try
	err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_index = ?",
		competition.ID, UserID, puzzleIndex-1).Find(&tries).Error
	if err != nil {
		return false
	}

	for _, try := range tries {
		if try.EndTime != nil {
			return true
		}
	}

	return false
}

func CheckRateLimit(try models.Try, config config.RateLimitConfig) (bool, time.Duration) {
	if try.LastMoveTime == nil {
		return false, 0
	}

	lastMove, err := time.Parse(time.RFC3339, *try.LastMoveTime)
	if err != nil {
		return false, 0
	}

	now := time.Now()
	if try.Attempts >= config.AttemptsThreshold2 {
		cooldownEnd := lastMove.Add(config.CooldownDuration2)
		if now.Before(cooldownEnd) {
			metrics.RateLimiterCooldowns.WithLabelValues("threshold2").Inc()
			return true, cooldownEnd.Sub(now)
		}
	} else if try.Attempts >= config.AttemptsThreshold1 {
		cooldownEnd := lastMove.Add(config.CooldownDuration1)
		if now.Before(cooldownEnd) {
			metrics.RateLimiterCooldowns.WithLabelValues("threshold1").Inc()
			return true, cooldownEnd.Sub(now)
		}
	}

	return false, 0
}