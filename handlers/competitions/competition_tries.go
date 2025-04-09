package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
    // Time constants for tries operations
    TriesOperationTimeout   = 10 * time.Second
    TriesCacheDuration      = 5 * time.Minute
    StatsCacheDuration      = 5 * time.Minute
    TriesCacheKeyPrefix     = "competition_tries:"
    UserTriesCacheKeyPrefix = "user_competition_tries:"
    StatsCacheKeyPrefix     = "competition_stats:"
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

// ExportCompetitionDataExcel exports competition data as an Excel file
// @Summary Export competition data as Excel
// @Description Export detailed competition data and leaderboard as an Excel file
// @Tags Competitions
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path string true "Competition ID"
// @Success 200 {file} file "Excel file download"
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/export [get]
// @Security Bearer
func ExportCompetitionDataExcel(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), TriesOperationTimeout)
    defer cancel()

    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    competitionID := c.Param("id")

    // Check if user has access to the competition
    if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    var competition models.Competition
    if err := database.DB.WithContext(ctx).First(&competition, "id = ?", competitionID).Error; err != nil {
        log.Printf("Competition not found (ID: %s): %v", competitionID, err)
        response.Error(c, http.StatusNotFound, ErrCompetitionNotFound)
        return
    }

    // Create a new Excel file
    f := excelize.NewFile()

    // Add Logs sheet with detailed competition data
    type LogEntry struct {
        Firstname    string  `gorm:"column:first_name"`
        Lastname     string  `gorm:"column:last_name"`
        Groups       string  `gorm:"column:groups"`
        PuzzleIndex  int     `gorm:"column:puzzle_index"`
        PuzzleLvl    string  `gorm:"column:puzzle_lvl"`
        Step         int     `gorm:"column:step"`
        StartTime    string  `gorm:"column:start_time"`
        LastMoveTime string  `gorm:"column:last_move_time"`
        EndTime      string  `gorm:"column:end_time"`
        LastAnswer   string  `gorm:"column:last_answer"`
        Attempts     int     `gorm:"column:attempts"`
        Score        float64 `gorm:"column:score"`
        Duration     string  `gorm:"column:duration"`
    }

    var logEntries []LogEntry
    logQuery := `
        SELECT
            u.firstname,
            u.lastname,
            STRING_AGG(g.name, ', ') AS groups,
            t.puzzle_index,
            t.puzzle_lvl,
            t.step,
            t.start_time,
            t.last_move_time,
            t.end_time,
            t.last_answer,
            t.attempts,
            t.score,
            CASE
                WHEN t.end_time IS NOT NULL THEN t.end_time - t.start_time
                ELSE NULL
            END AS duration
        FROM
            tries t
        JOIN
            users u ON t.user_id = u.id
        LEFT JOIN
            user_groups ug ON t.user_id = ug.user_id
        LEFT JOIN
            groups g ON ug.group_id = g.id
        WHERE
            t.competition_id = ?
        GROUP BY
            u.id, u.firstname, u.lastname, t.puzzle_index, t.puzzle_lvl, t.step,
            t.start_time, t.last_move_time, t.end_time, t.last_answer, t.attempts, t.score
        ORDER BY
            t.start_time DESC
    `

    if err := database.DB.WithContext(ctx).Raw(logQuery, competitionID).Scan(&logEntries).Error; err != nil {
        log.Printf("Failed to fetch logs data for competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch logs data")
        return
    }

    // Create Logs sheet
    sheetName := "Logs"
    f.NewSheet(sheetName)
    f.DeleteSheet("Sheet1") // Delete default sheet

    // Set headers for Logs sheet
    headers := []string{"First Name", "Last Name", "Groups", "Puzzle Index", "Puzzle Level", "Step", 
        "Start Time", "Last Move Time", "End Time", "Last Answer", "Attempts", "Score", "Duration"}
    
    for i, header := range headers {
        cell, _ := excelize.CoordinatesToCellName(i+1, 1)
        f.SetCellValue(sheetName, cell, header)
    }

    // Add log data
    for i, entry := range logEntries {
        row := i + 2 // Start from row 2 (after headers)
        f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.Firstname)
        f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Lastname)
        f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), entry.Groups)
        f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), entry.PuzzleIndex)
        f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.PuzzleLvl)
        f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), entry.Step)
        f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), entry.StartTime)
        f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), entry.LastMoveTime)
        f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), entry.EndTime)
        f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), entry.LastAnswer)
        f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), entry.Attempts)
        f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), entry.Score)
        f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), entry.Duration)
    }

    // Add Leaderboard sheet
    type LeaderboardEntry struct {
        UserID             string  `gorm:"column:user_id"`
        Firstname          string  `gorm:"column:firstname"`
        TotalScore         float64 `gorm:"column:total_score"`
        HighestPuzzleIndex int     `gorm:"column:highest_puzzle_index"`
        TotalAttempts      int     `gorm:"column:total_attempts"`
        FirstAction        string  `gorm:"column:first_action"`
        LastAction         string  `gorm:"column:last_action"`
    }

    var leaderboardEntries []LeaderboardEntry
    leaderboardQuery := `
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

    if err := database.DB.WithContext(ctx).Raw(leaderboardQuery, competitionID).Scan(&leaderboardEntries).Error; err != nil {
        log.Printf("Failed to fetch leaderboard data for competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch leaderboard data")
        return
    }

    // Create Leaderboard sheet
    leaderboardSheet := "Leaderboard"
    f.NewSheet(leaderboardSheet)

    // Set headers for Leaderboard sheet
    leaderboardHeaders := []string{"User ID", "First Name", "Total Score", "Highest Puzzle Index", 
        "Total Attempts", "First Action", "Last Action"}
    
    for i, header := range leaderboardHeaders {
        cell, _ := excelize.CoordinatesToCellName(i+1, 1)
        f.SetCellValue(leaderboardSheet, cell, header)
    }

    // Add leaderboard data
    for i, entry := range leaderboardEntries {
        row := i + 2 // Start from row 2 (after headers)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("A%d", row), entry.UserID)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("B%d", row), entry.Firstname)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("C%d", row), entry.TotalScore)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("D%d", row), entry.HighestPuzzleIndex)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("E%d", row), entry.TotalAttempts)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("F%d", row), entry.FirstAction)
        f.SetCellValue(leaderboardSheet, fmt.Sprintf("G%d", row), entry.LastAction)
    }

    // Set content type for Excel file
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-data-%s.xlsx", 
        competition.Title, time.Now().Format("2006-01-02")))

    // Write the Excel file to the response
    if err := f.Write(c.Writer); err != nil {
        log.Printf("Failed to generate Excel file: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to generate Excel file")
        return
    }
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
    ctx, cancel := context.WithTimeout(c.Request.Context(), TriesOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    competitionID := c.Param("id")
    if competitionID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition ID")
        return
    }
    
    // Check if we should use cached results
    isAdmin := hasCompetitionPermission(user, permissions.COMPETITIONS)
    var cacheKey string
    
    if isAdmin {
        cacheKey = TriesCacheKeyPrefix + competitionID + ":all"
    } else {
        cacheKey = TriesCacheKeyPrefix + competitionID + ":" + user.ID
    }
    
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedData != "" {
        var tries []models.Try
        if err := utils.UnmarshalJSON([]byte(cachedData), &tries); err == nil {
            c.JSON(http.StatusOK, tries)
            return
        }
        // If unmarshalling fails, continue with fetching from database
        log.Printf("Failed to unmarshal cached tries: %v", err)
    }

    // Check if user has access to the competition
    var competition models.Competition
    var tries []models.Try
    
    hasAccess := isAdmin
    if !hasAccess {
        if err := services.GetAccessibleCompetition(user.ID, competitionID, &competition); err == nil {
            hasAccess = true
        }
    }

    if !hasAccess {
        response.Error(c, http.StatusUnauthorized, ErrUnauthorizedAccess)
        return
    }

    // Execute the database query based on permissions
    if isAdmin {
        // Administrators can see all tries
        if err := database.DB.WithContext(ctx).
            Where("competition_id = ?", competitionID).
            Preload("User.Groups").
            Find(&tries).Error; err != nil {
            log.Printf("Failed to fetch competition tries: %v", err)
            response.Error(c, http.StatusInternalServerError, ErrFailedFetchCompetitionTries)
            return
        }
    } else {
        // Normal users can only see their own tries
        if err := database.DB.WithContext(ctx).
            Where("competition_id = ? AND user_id = ?", competitionID, user.ID).
            Preload("User.Groups").
            Find(&tries).Error; err != nil {
            log.Printf("Failed to fetch user competition tries: %v", err)
            response.Error(c, http.StatusInternalServerError, ErrFailedFetchCompetitionTries)
            return
        }
    }

    // Cache the results for future requests
    triesJSON, err := utils.MarshalJSON(tries)
    if err == nil {
        if err := database.REDIS.Set(ctx, cacheKey, string(triesJSON), TriesCacheDuration).Err(); err != nil {
            // Log error but continue with response
            log.Printf("Failed to cache competition tries: %v", err)
        }
    }

    c.JSON(http.StatusOK, tries)
}

// GetCompetitionTriesLdb retrieves all tries for a competition ready for leaderboard (removing any answers)
// @Summary Get all tries for a competition ready for leaderboard
// @Description Get all tries for the specified competition ready for leaderboard (removing any answers)
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {array} models.Try
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/tries/ldb [get]
// @Security Bearer
func GetCompetitionTriesLDB(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), TriesOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    competitionID := c.Param("id")
    if competitionID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition ID")
        return
    }
    
    // Check if user has access to the competition
    var competition models.Competition
    var tries []models.Try

    err = services.GetAccessibleCompetition(user.ID, competitionID, &competition)
    if err != nil {
        response.Error(c, http.StatusUnauthorized, ErrUnauthorizedAccess)
        return
    }

    // Execute the database query based on permissions
    if err := database.DB.WithContext(ctx).
        Where("competition_id = ?", competitionID).
        Preload("User.Groups").
        Find(&tries).Error; err != nil {
        log.Printf("Failed to fetch competition tries: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrFailedFetchCompetitionTries)
        return
    }

    // Remove answers from tries for leaderboard
    emptyString := ""
    for i := range tries {
        tries[i].LastAnswer = &emptyString
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
    ctx, cancel := context.WithTimeout(c.Request.Context(), TriesOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    competitionID := c.Param("id")
    if competitionID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition ID")
        return
    }

    // Try to get from cache first
    cacheKey := StatsCacheKeyPrefix + competitionID
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedData != "" {
        var stats CompetitionStatsResponse
        if err := utils.UnmarshalJSON([]byte(cachedData), &stats); err == nil {
            c.JSON(http.StatusOK, stats)
            return
        }
        // If unmarshalling fails, continue with calculation
        log.Printf("Failed to unmarshal cached statistics: %v", err)
    }

    // Check if user has access to the competition
    if !userHasAccessToCompetition(user.ID, competitionID) && 
       !hasCompetitionPermission(user, permissions.COMPETITIONS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    var competition models.Competition
    if err := database.DB.WithContext(ctx).First(&competition, "id = ?", competitionID).Error; err != nil {
        log.Printf("Competition not found (ID: %s): %v", competitionID, err)
        response.Error(c, http.StatusNotFound, ErrCompetitionNotFound)
        return
    }

    // Get user statistics with a single SQL query
    var userStats []UserStat
    query := `
        SELECT
            t.user_id,
            u.firstname,
            COALESCE(SUM(t.score), 0) AS total_score,
            MAX(t.puzzle_index) AS highest_puzzle_index,
            COALESCE(SUM(t.attempts), 0) AS total_attempts,
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

    if err := database.DB.WithContext(ctx).Raw(query, competitionID).Scan(&userStats).Error; err != nil {
        log.Printf("Failed to fetch statistics for competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch statistics")
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

    // Calculate averages
    var averageScore float64
    if activeUsers > 0 {
        averageScore = totalScore / float64(activeUsers)
    }

    // Create the response
    stats := CompetitionStatsResponse{
        CompetitionID: competitionID,
        Title:         competition.Title,
        TotalUsers:    totalUsers,
        ActiveUsers:   activeUsers,
        AverageScore:  averageScore,
        HighestScore:  highestScore,
        UserStats:     userStats[:min(10, len(userStats))], // Top 10 users
    }

    // Cache the statistics
    statsJSON, err := utils.MarshalJSON(stats)
    if err == nil {
        if err := database.REDIS.Set(ctx, cacheKey, string(statsJSON), StatsCacheDuration).Err(); err != nil {
            // Log error but continue with response
            log.Printf("Failed to cache competition statistics: %v", err)
        }
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
    ctx, cancel := context.WithTimeout(c.Request.Context(), TriesOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    competitionID := c.Param("id")
    targetUserID := c.Param("user_id")
    
    if competitionID == "" || targetUserID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition or user ID")
        return
    }

    // Check if user has permission to view others' tries
    isOwnTries := user.ID == targetUserID
    isAdmin := hasCompetitionPermission(user, permissions.COMPETITIONS)
    
    if !isOwnTries && !isAdmin {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionViewTries)
        return
    }
    
    // Try to get from cache first
    // cacheKey := UserTriesCacheKeyPrefix + competitionID + ":" + targetUserID
    // cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    // if err == nil && cachedData != "" {
    //     var tries []models.Try
    //     if err := utils.UnmarshalJSON([]byte(cachedData), &tries); err == nil {
    //         c.JSON(http.StatusOK, tries)
    //         return
    //     }
    //     // If unmarshalling fails, continue with fetching from database
    //     log.Printf("Failed to unmarshal cached user tries: %v", err)
    // }

    var tries []models.Try
    if err := database.DB.WithContext(ctx).
        Where("competition_id = ? AND user_id = ?", competitionID, targetUserID).
        Preload("User.Groups").
        Order("puzzle_index ASC, step ASC, start_time ASC").
        Find(&tries).Error; err != nil {
        log.Printf("Failed to fetch user tries: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch tries")
        return
    }

    // Cache the results for future requests
    // triesJSON, err := utils.MarshalJSON(tries)
    // if err == nil {
    //     if err := database.REDIS.Set(ctx, cacheKey, string(triesJSON), TriesCacheDuration).Err(); err != nil {
    //         // Log error but continue with response
    //         log.Printf("Failed to cache user tries: %v", err)
    //     }
    // }

    c.JSON(http.StatusOK, tries)
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}