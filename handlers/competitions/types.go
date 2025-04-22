package competitions

import "api/models"

// Constants for error messages
const (
	ErrCompetitionNotFound      = "Competition not found"
	ErrCatalogNotFound          = "Catalog not found"
	ErrGroupNotFound            = "Group not found"
	ErrNoPermissionView         = "User does not have permission to view competitions"
	ErrNoPermissionCreate       = "User does not haveCompetitionStatsResponse permission to create competitions"
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

type UserStat struct {
	UserID             string  `gorm:"column:user_id"`
	Firstname          string  `gorm:"column:first_name"`
	TotalScore         float64 `gorm:"column:total_score"`
	HighestPuzzleIndex int     `gorm:"column:highest_puzzle_index"`
	TotalAttempts      int     `gorm:"column:total_attempts"`
	FirstAction        string  `gorm:"column:first_action"`
	LastAction         string  `gorm:"column:last_action"`
}

// CreateCompetitionRequest model for creating a new competition
type CreateCompetitionRequest struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	CatalogID    string   `json:"catalog_id" binding:"required"`
	CatalogTheme string   `json:"catalog_theme" binding:"required"`
	GroupsIDs    []string `json:"groups_ids" binding:"required"`
	Show         bool     `json:"show"`
}

// UpdateCompetitionRequest model for updating a competition
type UpdateCompetitionRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	CatalogTheme    string   `json:"catalog_theme"`
	CatalogID       string   `json:"catalog_id"`
	GroupsIDs	   []string `json:"groups_ids"`
	Finished        *bool    `json:"finished"`
	Show            *bool    `json:"show"`
}

// CompetitionStatsResponse model for competition statistics
type CompetitionStatsResponse struct {
	CompetitionID   string `json:"competition_id"`
	Title           string `json:"title"`
	TotalUsers      int    `json:"total_users"`
	ActiveUsers     int    `json:"active_users"`
	AverageScore    float64 `json:"average_score"`
	HighestScore    float64 `json:"highest_score"`
	UserStats []UserStat `json:"user_statistics"`
}

// InputRequest model for input request
type InputRequest struct {
	CompetitionID   string  `json:"competition_id"`
	PuzzleID 	    string  `json:"puzzle_id"`
	PuzzleIndex 	int     `json:"puzzle_index"`
	PuzzleDifficulty string     `json:"puzzle_difficulty"`
}

// CompetitionTry model for competition try
type CompetitionTry struct {
	CompetitionID   string  `json:"competition_id"`
	PuzzleId 	    string  `json:"puzzle_id"`
	PuzzleIndex 	int     `json:"puzzle_index"`
	PuzzleStep 	    int     `json:"puzzle_step"`
	PuzzleDifficulty string     `json:"puzzle_difficulty"`
	UserID 		    string  `json:"user_id"`
	Answer        string  `json:"solution"`
}

// PuzzleTriesResponse model for tries with cooldown information
type PuzzleTriesResponse struct {
	Tries            []models.Try `json:"tries"`
	IsUnderCooldown  bool         `json:"is_under_cooldown"`
	CooldownRemaining int         `json:"cooldown_remaining_seconds"`
	LastTry          *models.Try  `json:"last_try,omitempty"`
}

// PuzzleAnswerResponse model for answering puzzles
type PuzzleAnswerResponse struct {
	IsCorrect        bool   `json:"is_correct"`
	PuzzleId         string `json:"puzzle_id"`
	PuzzleStep       int    `json:"puzzle_step"`
	IsUnderCooldown  bool   `json:"is_under_cooldown"`
	CooldownRemaining int   `json:"cooldown_remaining_seconds"`
}