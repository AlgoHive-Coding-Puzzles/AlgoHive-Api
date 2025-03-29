package services

import (
	"api/config"
	"api/database"
	"api/metrics"
	"api/models"
	"api/realtime"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var (
    // Define common errors as package-level variables for better error handling
    ErrTryAlreadyFinished = errors.New("try already finished")
    ErrFailedToFetchTry   = errors.New("failed to fetch try")
    ErrFailedToCreateTry  = errors.New("failed to create try")
    ErrFailedToUpdateTry  = errors.New("failed to update try")
)

// TriggerPuzzleFirstTry creates a new try or returns an existing one for step 1
func TriggerPuzzleFirstTry(competition models.Competition, puzzleID string, puzzleIndex int, puzzleLvl string, user models.User) (models.Try, error) {
    // Use a transaction to ensure atomicity and prevent race conditions
    tx := database.DB.Begin()
    if tx.Error != nil {
        return models.Try{}, fmt.Errorf("failed to begin transaction: %w", tx.Error)
    }
    
    // Improved rollback handling with defer
    defer func() {
        // Only rollback if we haven't committed
        if tx.Error != nil {
            tx.Rollback()
        }
    }()

    // Check if a try already exists with FOR UPDATE locking to prevent concurrent operations
    var existingTry models.Try
    err := tx.Set("gorm:query_option", "FOR UPDATE").
        Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
            competition.ID, user.ID, puzzleID, 1, puzzleIndex).
        First(&existingTry).Error

    if err == nil {
        // Try already exists, commit transaction and return it
        if err := tx.Commit().Error; err != nil {
            return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
        }
        return existingTry, nil
    } else if err != gorm.ErrRecordNotFound {
        // No need to explicitly rollback here as it'll be handled by defer
        return models.Try{}, fmt.Errorf("database error: %w", err)
    }

    // No try exists, create a new one
    now := time.Now().Format(time.RFC3339)
    newTry := models.Try{
        PuzzleID:      puzzleID,
        PuzzleIndex:   puzzleIndex,
        PuzzleLvl:     puzzleLvl,
        Step:          1,
        StartTime:     now,
        EndTime:       nil,
        Attempts:      0,
        LastAnswer:    nil,
        LastMoveTime:  nil,
        Score:         0,
        CompetitionID: competition.ID,
        UserID:        user.ID,
    }

    if err := tx.Create(&newTry).Error; err != nil {
        if config.Env == "development" {
            return models.Try{}, nil
        }
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToCreateTry, err)
    }

    // Commit the transaction before broadcasting
    if err := tx.Commit().Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Get user but with group association
    var userWithGroup models.User
    if err := database.DB.Preload("Groups").Where("id = ?", user.ID).First(&userWithGroup).Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to fetch user: %w", err)
    }
    newTry.User = &userWithGroup

    // Broadcast after successful DB transaction
    realtime.BroadcastTryUpdate(realtime.TryUpdate{
        CompetitionID: newTry.CompetitionID,
        Try:           newTry,
        UpdateType:    "new",
    })

    return newTry, nil
}

// GetPuzzleTry retrieves a specific try by its parameters
func GetPuzzleTry(competitionID string, puzzleID string, puzzleIndex int, step int, userID string) (models.Try, error) {
    var try models.Try
    if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
        competitionID, userID, puzzleID, step, puzzleIndex).First(&try).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return models.Try{}, fmt.Errorf("try not found")
        }
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToFetchTry, err)
    }

    return try, nil
}

// GetPuzzleFirstTry is a specific wrapper for step 1 tries
func GetPuzzleFirstTry(competitionID string, puzzleID string, puzzleIndex int, userID string) (models.Try, error) {
    return GetPuzzleTry(competitionID, puzzleID, puzzleIndex, 1, userID)
}

func UpdateTry(competition models.Competition, puzzleID string, puzzleIndex int, step int, user models.User, answer string) (models.Try, error) {
    // Use transaction for safe updates
    tx := database.DB.Begin()
    if tx.Error != nil {
        return models.Try{}, fmt.Errorf("failed to begin transaction: %w", tx.Error)
    }
    
    defer func() {
        if tx.Error != nil {
            tx.Rollback()
        }
    }()

    // Get try with row locking
    var existingTry models.Try
    if err := tx.Set("gorm:query_option", "FOR UPDATE").
        Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
        competition.ID, user.ID, puzzleID, step, puzzleIndex).First(&existingTry).Error; err != nil {
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToFetchTry, err)
    }

    if existingTry.EndTime != nil {
        tx.Rollback()
        return models.Try{}, ErrTryAlreadyFinished
    }

    // Update try
    moveTime := time.Now().Format(time.RFC3339)
    existingTry.LastMoveTime = &moveTime
    existingTry.LastAnswer = &answer
    existingTry.Attempts++
    
    if err := tx.Save(&existingTry).Error; err != nil {
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToUpdateTry, err)
    }

    if err := tx.Commit().Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Get user but with group association
    var userWithGroup models.User
    if err := database.DB.Preload("Groups").Where("id = ?", user.ID).First(&userWithGroup).Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to fetch user: %w", err)
    }

    existingTry.User = &userWithGroup

    // Broadcast after successful DB transaction
    realtime.BroadcastTryUpdate(realtime.TryUpdate{
        CompetitionID: existingTry.CompetitionID,
        Try:           existingTry,
        UpdateType:    "update",
    })

    return existingTry, nil
}

func EndTry(competition models.Competition, puzzleID string, puzzleIndex int, step int, user models.User, answer string) (models.Try, error) {
    // Use transaction for consistency
    tx := database.DB.Begin()
    if tx.Error != nil {
        return models.Try{}, fmt.Errorf("failed to begin transaction: %w", tx.Error)
    }
    
    defer func() {
        if tx.Error != nil {
            tx.Rollback()
        }
    }()

    // Get try with locking
    var existingTry models.Try
    if err := tx.Set("gorm:query_option", "FOR UPDATE").
        Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ? AND puzzle_index = ?",
        competition.ID, user.ID, puzzleID, step, puzzleIndex).First(&existingTry).Error; err != nil {
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToFetchTry, err)
    }

    if existingTry.EndTime != nil {
        tx.Rollback()
        return models.Try{}, ErrTryAlreadyFinished
    }

    // Update try
    now := time.Now().Format(time.RFC3339)
    existingTry.LastMoveTime = &now
    existingTry.LastAnswer = &answer
    existingTry.Attempts++
    existingTry.Score = 100 // Assuming score is set to 100 when ending the try
    existingTry.EndTime = &now
    
    if err := tx.Save(&existingTry).Error; err != nil {
        return models.Try{}, fmt.Errorf("%w: %s", ErrFailedToUpdateTry, err)
    }

    // Create next step try if needed (within the same transaction)
    var newTry *models.Try
    if step == 1 {
        nextTry := models.Try{
            PuzzleID:      puzzleID,
            PuzzleIndex:   existingTry.PuzzleIndex,
            PuzzleLvl:     existingTry.PuzzleLvl,
            Step:          2,
            StartTime:     now,
            EndTime:       nil,
            Attempts:      0,
            LastAnswer:    nil,
            LastMoveTime:  nil,
            Score:         0,
            CompetitionID: competition.ID,
            UserID:        user.ID,
        }
        
        if err := tx.Create(&nextTry).Error; err != nil {
            return models.Try{}, fmt.Errorf("failed to create next step try: %w", err)
        }
        newTry = &nextTry
    }

    if err := tx.Commit().Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Get user but with group association
    var userWithGroup models.User
    if err := database.DB.Preload("Groups").Where("id = ?", user.ID).First(&userWithGroup).Error; err != nil {
        return models.Try{}, fmt.Errorf("failed to fetch user: %w", err)
    }

    // Associate user after transaction commit
    existingTry.User = &userWithGroup

    // Broadcast completed try
    realtime.BroadcastTryUpdate(realtime.TryUpdate{
        CompetitionID: existingTry.CompetitionID,
        Try:           existingTry,
        UpdateType:    "update",
    })

    // Broadcast new try if created
    if newTry != nil {
        newTry.User = &user
        realtime.BroadcastTryUpdate(realtime.TryUpdate{
            CompetitionID: newTry.CompetitionID,
            Try:           *newTry,
            UpdateType:    "new",
        })
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

// UserHasPermissionToViewPuzzle checks if a user can view a specific puzzle
// The User has permission to view the puzzle if:
// 1. The puzzleIndex is 0
// 2. Or if the puzzle[puzzleIndex-1] is already solved (puzzle solved means that the try has end_time != nil)
func UserHasPermissionToViewPuzzle(competition models.Competition, puzzleIndex int, userID string) bool {
    if puzzleIndex == 0 {
        return true
    }

    // Check if previous puzzle is completed - more efficient query
    var count int64
    err := database.DB.Model(&models.Try{}).
        Where("competition_id = ? AND user_id = ? AND puzzle_index = ? AND end_time IS NOT NULL",
            competition.ID, userID, puzzleIndex-1).
        Count(&count).Error
    
    return err == nil && count > 0
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
    
    // Check highest threshold first for efficiency
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