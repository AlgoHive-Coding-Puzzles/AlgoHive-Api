package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"api/utils/permissions"
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

    // Get user statistics with a single SQL query
    type UserStat struct {
        UserID            string  `gorm:"column:user_id"`
        Firstname         string  `gorm:"column:firstname"`
        TotalScore        float64 `gorm:"column:total_score"`
        HighestPuzzleIndex int    `gorm:"column:highest_puzzle_index"`
        TotalAttempts     int     `gorm:"column:total_attempts"`
        FirstAction       string  `gorm:"column:first_action"`
        LastAction        string  `gorm:"column:last_action"`
    }

    var userStats []UserStat
    query := `
        SELECT
            t.user_id,
            u.firstname,
            SUM(t.score) AS total_score,
            MAX(t.puzzle_index) AS highest_puzzle_index,
            SUM(t.attempts) AS total_attempts,
            MIN(t.start_time) AS first_action,
            MAX(COALESCE(t.end_time, t.last_move_time)) AS last_action
        FROM
            tries t
        JOIN
            users u ON t.user_id = u.id
        WHERE
            t.competition_id = ?
        GROUP BY
            t.user_id, u.firstname
        ORDER BY
            total_score DESC,
            highest_puzzle_index DESC,
            first_action ASC
    `

    if err := database.DB.Raw(query, competitionID).Scan(&userStats).Error; err != nil {
        respondWithError(c, http.StatusInternalServerError, "Failed to fetch statistics")
        return
    }

    // Calculate overall statistics from the user data
    totalUsers := len(userStats)
    activeUsers := 0
    var totalScore float64
    var highestScore float64

    for _, stat := range userStats {
        if stat.TotalScore > 0 {
            activeUsers++
            totalScore += stat.TotalScore
            if stat.TotalScore > highestScore {
                highestScore = stat.TotalScore
            }
        }
    }

    // Calculate averages and rates
    var averageScore float64

    if totalUsers > 0 {
        if activeUsers > 0 {
            averageScore = totalScore / float64(activeUsers)
        }
    }

    // Create the response
    stats := CompetitionStatsResponse{
        CompetitionID:  competitionID,
        Title:          competition.Title,
        TotalUsers:     totalUsers,
        ActiveUsers:    activeUsers,
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