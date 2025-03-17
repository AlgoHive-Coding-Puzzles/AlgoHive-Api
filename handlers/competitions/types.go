package competitions

import (
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

// respondWithError envoie une réponse d'erreur standardisée
func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
