package services

import (
	"api/database"
	"api/models"
	"fmt"
	"time"
)

func TriggerPuzzleFirstTry(competition models.Competition, puzzleID string, puzzleIndex int, puzzleLvl string, userID string) (models.Try, error) {
	// Step 1: Check if a try already exists (avoid raising an error if it doesn't)
	var exists bool
	err := database.DB.Model(&models.Try{}).
		Select("1").
		Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
			competition.ID, userID, puzzleID, 1, puzzleIndex).
		Limit(1).
		Scan(&exists).Error
	
	if err != nil {
		return models.Try{}, fmt.Errorf("database error: %w", err)
	}


	if exists {
		// Step 2.A - Return the existing try if it exists
		var existingTry models.Try
		if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ?",
		competition.ID, userID, puzzleID, 1).First(&existingTry).Error; err != nil {
			return models.Try{}, fmt.Errorf("failed to fetch existing try: %w", err)
		}

		return existingTry, nil
	} else {
		// Step 2.B - Create a new try if it doesn't exist
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

		if err := database.DB.Create(&newTry).Error; err != nil {
			return models.Try{}, fmt.Errorf("failed to create new try: %w", err)
		}

		return newTry, nil
	}
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