package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"api/utils/permissions"
	"log"
	"net/http"

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
	var competition models.Competition
	if err := services.GetAccessibleCompetition(user.ID, competitionID, &competition); err == nil || hasCompetitionPermission(user, permissions.COMPETITIONS) {
	// Administrators can see all tries
	log.Printf("User %s has access to competition %s", user.ID, competitionID)
		if err := database.DB.Where("competition_id = ?", competitionID).
			Preload("User.Groups").Find(&tries).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
			return
		}
	} else {
		// Normal users can only see their own tries
		if err := database.DB.Where("competition_id = ? AND user_id = ?", 
			competitionID, user.ID).Preload("User.Groups").Find(&tries).Error; err != nil {
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
	if user.ID != targetUserID && !hasCompetitionPermission(user, permissions.COMPETITIONS)  {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionViewTries)
		return
	}

	var tries []models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ?", 
		competitionID, targetUserID).Preload("User.Groups").Find(&tries).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
		return
	}

	c.JSON(http.StatusOK, tries)
}