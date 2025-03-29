package competitions

// Constants for error messages
const (
	ErrCompetitionNotFound      = "Competition not found"
	ErrGroupNotFound            = "Group not found"
	ErrCatalogNotFound          = "API environment not found"
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

// CreateCompetitionRequest model for creating a competition
type CreateCompetitionRequest struct {
	Title           string   `json:"title" binding:"required"`
	Description     string   `json:"description" binding:"required"`
	CatalogTheme    string   `json:"catalog_theme" binding:"required"`
	CatalogID       string   `json:"catalog_id" binding:"required"`
	GroupIds        []string `json:"group_ids"`
	Show            bool     `json:"show"`
}

// UpdateCompetitionRequest model for updating a competition
type UpdateCompetitionRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	CatalogTheme    string   `json:"catalog_theme"`
	CatalogID       string   `json:"catalog_id"`
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