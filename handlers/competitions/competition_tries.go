package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"api/utils/permissions"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateTryRequest model for creating a try
type CreateTryRequest struct {
	PuzzleID    string `json:"puzzle_id" binding:"required"`
	PuzzleIndex int    `json:"puzzle_index" binding:"required"`
	PuzzleLvl   string `json:"puzzle_lvl" binding:"required"`
	Step        int    `json:"step" binding:"required"`
}

// UpdateTryRequest model for updating a try
type UpdateTryRequest struct {
	EndTime  string  `json:"end_time" binding:"required"`
	Attempts int     `json:"attempts" binding:"required"`
	Score    float64 `json:"score" binding:"required"`
}

// StartCompetitionTry starts a try for a competition
// @Summary Start a competition try
// @Description Start a new try for a puzzle in a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param try body CreateTryRequest true "Try details"
// @Success 201 {object} models.Try
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/tries [post]
// @Security Bearer
func StartCompetitionTry(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")
	
	// Check if user has access to the competition
	if !userHasAccessToCompetition(user.ID, competitionID) && !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var competition models.Competition
	if err := database.DB.First(&competition, "id = ?", competitionID).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Check if competition is finished
	if competition.Finished {
		respondWithError(c, http.StatusBadRequest, "Competition is already finished")
		return
	}

	var req CreateTryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Create a new try
	now := time.Now()
	try := models.Try{
		PuzzleID:      req.PuzzleID,
		PuzzleIndex:   req.PuzzleIndex,
		PuzzleLvl:     req.PuzzleLvl,
		Step:          req.Step,
		StartTime:     now.Format(time.RFC3339),
		Attempts:      0,
		Score:         0,
		CompetitionID: competitionID,
		UserID:        user.ID,
	}

	if err := database.DB.Create(&try).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to create try")
		return
	}

	c.JSON(http.StatusCreated, try)
}

// FinishCompetitionTry finishes a try for a competition
// @Summary Finish a competition try
// @Description Complete an ongoing try for a puzzle in a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param try_id path string true "Try ID"
// @Param try body UpdateTryRequest true "Try details"
// @Success 200 {object} models.Try
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/tries/{try_id} [put]
// @Security Bearer
func FinishCompetitionTry(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")
	tryID := c.Param("try_id")

	var try models.Try
	if err := database.DB.Where("id = ? AND competition_id = ? AND user_id = ?", 
		tryID, competitionID, user.ID).First(&try).Error; err != nil {
		respondWithError(c, http.StatusNotFound, "Try not found")
		return
	}

	// Check if try is already finished
	if try.EndTime != nil {
		respondWithError(c, http.StatusBadRequest, "Try is already finished")
		return
	}

	var req UpdateTryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Update the try
	try.EndTime = &req.EndTime
	try.Attempts = req.Attempts
	try.Score = req.Score

	if err := database.DB.Save(&try).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update try")
		return
	}

	c.JSON(http.StatusOK, try)
}

// GetCompetitionTries retrieves all tries for a competition
// @Summary Get all tries for a competition
// @Description Get all tries for the specified competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {array} models.Try
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/tries [get]
// @Security Bearer
func GetCompetitionTries(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")

	// Check if user has permission to see all tries or only their own
	var tries []models.Try
	if hasCompetitionPermission(user, permissions.COMPETITIONS) {
		// Administrators can see all tries
		if err := database.DB.Where("competition_id = ?", competitionID).
			Preload("User").Find(&tries).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
			return
		}
	} else {
		// Normal users can only see their own tries
		if err := database.DB.Where("competition_id = ? AND user_id = ?", 
			competitionID, user.ID).Find(&tries).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
			return
		}
	}

	c.JSON(http.StatusOK, tries)
}

// GetCompetitionStatistics retrieves statistics for a competition
// @Summary Get competition statistics
// @Description Get statistics for the specified competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {object} CompetitionStatsResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/statistics [get]
// @Security Bearer
func GetCompetitionStatistics(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")

	// Check if user has access to the competition
	if !userHasAccessToCompetition(user.ID, competitionID) && !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var competition models.Competition
	if err := database.DB.First(&competition, "id = ?", competitionID).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Calculate statistics
	var totalUsers int64
	var activeUsers int64
	var completionRate float64
	var averageScore float64
	var highestScore float64

	// Total number of users who have at least one try
	database.DB.Model(&models.Try{}).
		Select("COUNT(DISTINCT user_id)").
		Where("competition_id = ?", competitionID).
		Count(&totalUsers)

	// Number of active users (who have at least one completed try)
	database.DB.Model(&models.Try{}).
		Select("COUNT(DISTINCT user_id)").
		Where("competition_id = ? AND end_time IS NOT NULL", competitionID).
		Count(&activeUsers)

	// Calculate completion rate, average score, and highest score
	if totalUsers > 0 {
		completionRate = float64(activeUsers) / float64(totalUsers) * 100
		
		// Average score
		database.DB.Model(&models.Try{}).
			Select("COALESCE(AVG(score), 0)").
			Where("competition_id = ? AND end_time IS NOT NULL", competitionID).
			Scan(&averageScore)
		
		// Highest score
		database.DB.Model(&models.Try{}).
			Select("COALESCE(MAX(score), 0)").
			Where("competition_id = ? AND end_time IS NOT NULL", competitionID).
			Scan(&highestScore)
	}

	stats := CompetitionStatsResponse{
		CompetitionID:  competitionID,
		Title:          competition.Title,
		TotalUsers:     int(totalUsers),
		ActiveUsers:    int(activeUsers),
		CompletionRate: completionRate,
		AverageScore:   averageScore,
		HighestScore:   highestScore,
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserCompetitionTries retrieves all tries for a user in a competition
// @Summary Get user tries for a competition
// @Description Get all tries for a specific user in the specified competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param user_id path string true "User ID"
// @Success 200 {array} models.Try
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/users/{user_id}/tries [get]
// @Security Bearer
func GetUserCompetitionTries(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")
	targetUserID := c.Param("user_id")

	// Check if user has permission to view others' tries
	if user.ID != targetUserID && !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionViewTries)
		return
	}

	var tries []models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ?", 
		competitionID, targetUserID).Find(&tries).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
		return
	}

	c.JSON(http.StatusOK, tries)
}

// [POST] NewCompetitionTry creates or updates a try for a competition and return if the try solution is correct
// @Summary Create or update a competition try
// @Description Create or update a try for a competition and return if the try solution is correct
// @Tags Competitions
// @Accept json
// @Produce json
// @Param try body CompetitionTry true "Competition try"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/tries [post]
// @Security Bearer
func NewCompetitionTry(c *gin.Context) {
    // Step 1: Authenticate the user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Step 2: Parse the request body
    var req CompetitionTry
    if err := c.ShouldBindJSON(&req); err != nil {
        respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
        return
    }

    // Step 3: Validate user access to the competition
    competition, err := FetchCompetition(req.UserID, req.CompetitionID)
    if err != nil {
        respondWithError(c, http.StatusForbidden, err.Error())
        return
    }

    // Step 4: Validate the try step and construct the API URL
    stepUrl, err := GetStepURL(req.Step)
    if err != nil {
        respondWithError(c, http.StatusBadRequest, err.Error())
        return
    }

    apiURL := ConstructAPIURL(competition, req, stepUrl)

    // Step 5: Check the solution via the external service
    result, err := services.CatalogProxyGet(apiURL)
    if err != nil || result == nil || result["matches"] == nil {
        respondWithError(c, http.StatusInternalServerError, "Failed to check solution")
        return
    }

    isCorrect := result["matches"].(bool)

    // Step 6: Handle the try (create or update)
    if err := handleTry(user.ID, req, isCorrect); err != nil {
        respondWithError(c, http.StatusInternalServerError, err.Error())
        return
    }

    // Step 7: Respond with the result
    c.JSON(http.StatusOK, gin.H{
        "matches": isCorrect,
    })
}

// handleTry creates or updates a try based on the request and result
func handleTry(userID string, req CompetitionTry, isCorrect bool) error {
    var existingTry models.Try

    // Check if a try already exists
    err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND step = ?",
        req.CompetitionID, userID, req.PuzzleId, req.Step).First(&existingTry).Error

    if err == nil && existingTry.ID != "" {
        // Update the existing try
        if existingTry.EndTime != nil {
            return fmt.Errorf("try already finished")
        }
        existingTry.LastAnswer = req.Solution
        existingTry.Attempts++
        if isCorrect {
            existingTry.Score++
            now := time.Now().Format(time.RFC3339)
            existingTry.EndTime = &now
        }
        return database.DB.Save(&existingTry).Error
    }

    // Create a new try
    newTry := models.Try{
        PuzzleID:      req.PuzzleId,
        PuzzleIndex:   req.PuzzleIndex,
        PuzzleLvl:     req.PuzzleDifficulty,
        Step:          req.Step,
        StartTime:     time.Now().Format(time.RFC3339),
        EndTime:       nil,
        Attempts:      1,
        LastAnswer:    req.Solution,
        Score:         0,
        CompetitionID: req.CompetitionID,
        UserID:        userID,
    }

    if isCorrect {
        newTry.Score = 1
        now := time.Now().Format(time.RFC3339)
        newTry.EndTime = &now
    }

    return database.DB.Create(&newTry).Error
}