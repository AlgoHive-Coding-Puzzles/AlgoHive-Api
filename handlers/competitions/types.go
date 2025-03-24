package competitions

import (
	"api/database"
	"api/models"
	"api/services"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

// Constantes pour les messages d'erreur
const (
	ErrCompetitionNotFound      = "Competition not found"
	ErrGroupNotFound            = "Group not found"
	ErrCatalogNotFound          = "API environment not found"
	ErrNoPermissionView         = "User does not have permission to view competitions"
	ErrNoPermissionCreate       = "User does not have permission to create competitions"
	ErrNoPermissionUpdate       = "User does not have permission to update competitions"
	ErrNoPermissionDelete       = "User does not have permission to delete competitions"
	ErrNoPermissionManageGroups = "User does not have permission to manage competition groups"
	ErrNoPermissionViewTries    = "User does not have permission to view competition tries"
	ErrFailedFetchCompetitions  = "Failed to fetch competitions"
	ErrFailedFetchCompetitionTries = "Failed to fetch competition tries"
	ErrFailedCreateCompetition  = "Failed to create competition"
	ErrFailedUpdateCompetition  = "Failed to update competition"
	ErrFailedDeleteCompetition  = "Failed to delete competition"
	ErrInvalidRequest           = "Invalid request data"
	ErrFailedAddGroup           = "Failed to add group to competition"
	ErrFailedRemoveGroup        = "Failed to remove group from competition"
	ErrNoPermissionFinish	  = "User does not have permission to finish competitions"
	ErrFailedToggleFinished	  = "Failed to toggle competition finished status"
	ErrNoPermissionVisibility	  = "User does not have permission to change competition visibility"
	ErrFailedToggleVisibility	  = "Failed to toggle competition visibility"
	ErrUnauthorizedAccess	  = "Unauthorized access to competition"
)

// CreateCompetitionRequest modèle pour créer une compétition
type CreateCompetitionRequest struct {
	Title           string   `json:"title" binding:"required"`
	Description     string   `json:"description" binding:"required"`
	CatalogTheme    string   `json:"catalog_theme" binding:"required"`
	CatalogID       string   `json:"catalog_id" binding:"required"`
	GroupIds        []string `json:"group_ids"`
	Show            bool     `json:"show"`
}

// UpdateCompetitionRequest modèle pour mettre à jour une compétition
type UpdateCompetitionRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	CatalogTheme    string   `json:"catalog_theme"`
	CatalogID       string   `json:"catalog_id"`
	Finished        *bool    `json:"finished"`
	Show            *bool    `json:"show"`
}

// CompetitionStatsResponse modèle pour les statistiques d'une compétition
type CompetitionStatsResponse struct {
	CompetitionID   string `json:"competition_id"`
	Title           string `json:"title"`
	TotalUsers      int    `json:"total_users"`
	ActiveUsers     int    `json:"active_users"`
	CompletionRate  float64 `json:"completion_rate"`
	AverageScore    float64 `json:"average_score"`
	HighestScore    float64 `json:"highest_score"`
}

type CompetitionTry struct {
	CompetitionID   string  `json:"competition_id"`
	Theme 		    string  `json:"theme"`
	PuzzleId 	    string  `json:"puzzle_id"`
	PuzzleIndex 	int     `json:"puzzle_index"`
	PuzzleDifficulty string     `json:"puzzle_difficulty"`
	Step 		    int     `json:"step"`
	UserID 		    string  `json:"user_id"`
	Solution        string  `json:"solution"`
}

// respondWithError envoie une réponse d'erreur standardisée
func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// getStepURL validates the step and returns the corresponding URL
func GetStepURL(step int) (string, error) {
    switch step {
    case 1:
        return "/first", nil
    case 2:
        return "/second", nil
    default:
        return "", fmt.Errorf("invalid step")
    }
}

// constructAPIURL builds the API URL for checking the solution
func ConstructAPIURL(competition models.Competition, req CompetitionTry, stepUrl string) string {
	address, err := services.GetAddressFromCatalogId(competition.CatalogID)
	if err != nil {
		return ""
	}

    return fmt.Sprintf("%s/puzzle/check%s?theme=%s&puzzle=%s&unique_id=%s&solution=%s",
		address,
        stepUrl,
        competition.CatalogTheme,
        req.PuzzleId,
        req.UserID,
        req.Solution,
    )
}

// fetchCompetition validates user access to the competition
func FetchCompetition(userID, competitionID string) (models.Competition, error) {
    var competition models.Competition
    err := database.DB.Raw(`
        SELECT DISTINCT c.*
        FROM competitions c
        JOIN competition_groups cat ON c.id = cat.competition_id
        JOIN user_groups ug ON cat.group_id = ug.group_id
        WHERE ug.user_id = ? AND c.id = ? AND c.show = true
    `, userID, competitionID).Scan(&competition).Error

	log.Printf("Executing query: SELECT DISTINCT c.* FROM competitions c JOIN competition_groups cat ON c.id = cat.competition_id JOIN user_groups ug ON cat.group_id = ug.group_id WHERE ug.user_id = '%s' AND c.id = '%s' AND c.show = true", userID, competitionID)
	log.Println("Competition ID:", competitionID)
	log.Println("User ID:", userID)
	log.Println("Query Error:", err)
	log.Println("Competition Data:", competition)

    if err != nil || competition.ID == "" {
        return competition, fmt.Errorf("user does not have access to this competition")
    }
    return competition, nil
}