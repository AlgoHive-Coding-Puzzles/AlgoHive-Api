package competitions

import (
	"api/config"
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"api/utils"
	"api/utils/response"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
    CompetitionTimeout        = 30 * time.Second
    PuzzleInputCacheKeyPrefix = "comp_puzzle_input:"
    PuzzleInputCacheDuration  = 30 * time.Minute
)

// GetInputFromCompetition gets input for a puzzle from a competition
// @Summary Get input from a competition, and create a new try if needed
// @Description Get input from a competition, and create a new try if needed
// @Tags Competitions
// @Accept json
// @Produce json
// @Param inputRequest body InputRequest true "Input request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/input [post]
// @Security Bearer
func GetInputFromCompetition(c *gin.Context) {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), CompetitionTimeout)
    defer cancel()

    // Step 1: Authenticate the user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Step 2: Parse the request body
    var inputRequest InputRequest
    if err := c.ShouldBindJSON(&inputRequest); err != nil {
        response.Error(c, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Validate input request
    if inputRequest.CompetitionID == "" || inputRequest.PuzzleID == "" {
        response.Error(c, http.StatusBadRequest, "Missing required fields")
        return
    }

    // Step 3: Get the target Competition
    var competition models.Competition
    err = services.GetAccessibleCompetition(user.ID, inputRequest.CompetitionID, &competition)
    if err != nil {
        log.Printf("Access denied to competition %s for user %s: %v", 
            inputRequest.CompetitionID, user.ID, err)
        response.Error(c, http.StatusForbidden, err.Error())
        return
    }

    // Step 4: Create a new try if it's needed
    if _, err := services.TriggerPuzzleFirstTry(competition, 
        inputRequest.PuzzleID, 
        inputRequest.PuzzleIndex, 
        inputRequest.PuzzleDifficulty, 
        user); err != nil {
        log.Printf("Failed to create first try: %v", err)
        // Continue anyway, this is not critical
    }

    // Try to get puzzle input from cache first
    cacheKey := PuzzleInputCacheKeyPrefix + 
        competition.CatalogID + ":" + 
        competition.CatalogTheme + ":" + 
        inputRequest.PuzzleID + ":" + 
        user.ID

    var puzzleInput interface{}
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedData != "" {
        // Cache hit - use cached result
        if err := utils.UnmarshalJSON([]byte(cachedData), &puzzleInput); err == nil {
            c.JSON(http.StatusOK, puzzleInput)
            return
        }
        // If unmarshalling fails, continue to fetch from API
    }

    // Step 5: Get the puzzle input
    puzzleInput, err = services.GetPuzzleInput(
        competition.CatalogID,
        competition.CatalogTheme,
        inputRequest.PuzzleID,
        user.ID,
        ctx,
    )
    
    if err != nil {
        log.Printf("Failed to get puzzle input: %v", err)
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    // Cache the puzzle input
    inputJSON, err := utils.MarshalJSON(puzzleInput)
    if err == nil {
        if err := database.REDIS.Set(ctx, cacheKey, string(inputJSON), PuzzleInputCacheDuration).Err(); err != nil {
            log.Printf("Failed to cache puzzle input: %v", err)
            // Continue anyway, this is not critical
        }
    }

    c.JSON(http.StatusOK, puzzleInput)
}

// AnswerPuzzle processes a puzzle answer submission
// @Summary Answer a part of a puzzle
// @Description Answer a part of a puzzle
// @Tags Competitions
// @Accept json
// @Produce json
// @Param competitionTry body CompetitionTry true "Competition try"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 429 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /competitions/answer_puzzle [post]
// @Security Bearer
func AnswerPuzzle(c *gin.Context) {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), CompetitionTimeout)
    defer cancel()

    // Step 1: Authenticate the user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Step 2: Parse the request body
    var req CompetitionTry
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, ErrInvalidRequest)
        return
    }

    // Validate request
    if req.CompetitionID == "" || req.PuzzleId == "" {
        response.Error(c, http.StatusBadRequest, "Missing required fields")
        return
    }

    // Step 3: Validate user access to the competition
    var competition models.Competition
    err = services.GetAccessibleCompetition(user.ID, req.CompetitionID, &competition)
    if err != nil {
        log.Printf("Access denied to competition %s for user %s: %v", 
            req.CompetitionID, user.ID, err)
        response.Error(c, http.StatusForbidden, err.Error())
        return
    }

    // Step 4: Get the existing try and check rate limits
    existingTry, err := services.GetPuzzleTry(
        competition.ID, 
        req.PuzzleId, 
        req.PuzzleIndex, 
        req.PuzzleStep, 
        user.ID,
    )
    
    if err != nil {
        if err.Error() == "record not found" {
            log.Printf("Try not found for puzzle %s in competition %s", 
                req.PuzzleId, req.CompetitionID)
            response.Error(c, http.StatusBadRequest, "Try not found")
            return
        }
        log.Printf("Error fetching try: %v", err)
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    // Check rate limits
    isLimited, remainingTime := services.CheckRateLimit(existingTry, config.DefaultRateLimitConfig)

    if isLimited {
        c.JSON(http.StatusTooManyRequests, gin.H{
            "error": "Rate limit exceeded",
            "wait_time_seconds": int(remainingTime.Seconds()),
        })
        return
    }

    // Step 5: Validate the try
    isCorrect, err := services.CheckPuzzleAnswer(
        competition.CatalogID, 
        competition.CatalogTheme, 
        req.PuzzleId, 
        req.PuzzleStep, 
        user.ID, 
        req.Answer,
    )
    
    if err != nil {
        log.Printf("Failed to check solution: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to check solution")
        return
    }

    // Step 6: Handle the try (create or update)
    if !isCorrect {
        _, err := services.UpdateTry(
            competition, 
            req.PuzzleId, 
            req.PuzzleIndex, 
            req.PuzzleStep, 
            user, 
            req.Answer,
        )
        
        if err != nil {
            log.Printf("Failed to update try: %v", err)
            response.Error(c, http.StatusInternalServerError, "Failed to update try")
            return
        }
        
        // Log unsuccessful attempt
        log.Printf("Incorrect answer for puzzle %s by user %s", req.PuzzleId, user.ID)
    } else {
        _, err := services.EndTry(
            competition, 
            req.PuzzleId, 
            req.PuzzleIndex, 
            req.PuzzleStep, 
            user, 
            req.Answer,
        )
        
        if err != nil {
            log.Printf("Failed to end try: %v", err)
            response.Error(c, http.StatusInternalServerError, "Failed to end try")
            return
        }
        
        // Log successful completion
        log.Printf("Correct answer for puzzle %s by user %s", req.PuzzleId, user.ID)
        
        // Invalidate any cached puzzle input
        cacheKey := PuzzleInputCacheKeyPrefix + 
            competition.CatalogID + ":" + 
            competition.CatalogTheme + ":" + 
            req.PuzzleId + ":" + 
            user.ID
            
        if err := database.REDIS.Del(ctx, cacheKey).Err(); err != nil {
            // Just log error, don't fail the request
            log.Printf("Failed to invalidate puzzle input cache: %v", err)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "is_correct": isCorrect,
        "puzzle_id": req.PuzzleId,
        "puzzle_step": req.PuzzleStep,
    })
}

// GetTriesFromCompetitonPuzzle fetches tries for a specific puzzle
// @Summary Get all tries for a specific puzzle in a competition
// @Description Get all tries for a specific puzzle in a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param comptition_id path string true "Competition ID"
// @Param puzzle_id path string true "Puzzle ID"
// @Param puzzle_index path string true "Puzzle index"
// @Success 200 {array} models.Try
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/{comptition_id}/puzzles/{puzzle_id}/{puzzle_index}/tries [get]
// @Security Bearer
func GetTriesFromCompetitonPuzzle(c *gin.Context) {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), CompetitionTimeout)
    defer cancel()
    
    competitionID := c.Param("id")
    puzzleID := c.Param("puzzle_id")
    puzzleIndex := c.Param("puzzle_index")

    if competitionID == "" || puzzleID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition or puzzle ID")
        return
    }

    // Step 1: Authenticate the user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Step 2: Get the target Competition
    var competition models.Competition
    err = services.GetAccessibleCompetition(user.ID, competitionID, &competition)
    if err != nil {
        log.Printf("Access denied to competition %s for user %s: %v", 
            competitionID, user.ID, err)
        response.Error(c, http.StatusForbidden, err.Error())
        return
    }

    // Step 3: Get the tries for the puzzle with context
    var tries []models.Try
    if err := database.DB.WithContext(ctx).
        Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND puzzle_index = ?",
        competitionID, user.ID, puzzleID, puzzleIndex).
        Order("start_time ASC").
        Find(&tries).Error; err != nil {
        log.Printf("Failed to fetch tries: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch tries")
        return
    }

    c.JSON(http.StatusOK, tries)
}

// UserHasPermissionToViewPuzzle checks user's permissions
// @Summary Check if the user has permission to view a puzzle
// @Description Check if the user has permission to view a puzzle
// @Tags Competitions
// @Accept json
// @Produce json
// @Param competition_id path string true "Competition ID"
// @Param puzzle_index path string true "Puzzle index"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/{id}/puzzles/{puzzle_index}/permission [get]
// @Security Bearer
func UserHasPermissionToViewPuzzle(c *gin.Context) {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), CompetitionTimeout)
    defer cancel()
    
    competitionID := c.Param("id")
    puzzleIndex := c.Param("puzzle_index")

    if competitionID == "" || puzzleIndex == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition or puzzle index")
        return
    }

    // Step 1: Authenticate the user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Step 2: Get the target Competition with context
    var competition models.Competition
    err = services.GetAccessibleCompetition(user.ID, competitionID, &competition)
    if err != nil {
        log.Printf("Access denied to competition %s for user %s: %v", 
            competitionID, user.ID, err)
        response.Error(c, http.StatusForbidden, err.Error())
        return
    }

    // Step 3: Check if the user has permission to view the puzzle
    puzzleIndexInt, err := strconv.Atoi(puzzleIndex)
    if err != nil {
        response.Error(c, http.StatusBadRequest, "Invalid puzzle index")
        return
    }
    
    // Cache key for permission check
    permCacheKey := "perm_check:" + competitionID + ":" + user.ID + ":" + puzzleIndex
    cachedPerm, err := database.REDIS.Get(ctx, permCacheKey).Result()
    if err == nil && cachedPerm != "" {
        // Use cached permission result
        hasPermission := cachedPerm == "true"
        c.JSON(http.StatusOK, gin.H{"has_permission": hasPermission})
        return
    }
    
    // Calculate permission from DB
    hasPermission := services.UserHasPermissionToViewPuzzle(competition, puzzleIndexInt, user.ID)
    
    // Cache the permission result (short TTL since permissions can change)
    permValue := "false"
    if hasPermission {
        permValue = "true"
    }
    
    if err := database.REDIS.Set(ctx, permCacheKey, permValue, 5*time.Minute).Err(); err != nil {
        // Just log error, don't fail the request
        log.Printf("Failed to cache permission check: %v", err)
    }

    c.JSON(http.StatusOK, gin.H{"has_permission": hasPermission})
}