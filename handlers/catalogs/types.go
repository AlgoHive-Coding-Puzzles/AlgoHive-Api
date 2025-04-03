package catalogs

// Error message constants
const (
    ErrCatalogNotFound      = "API not found"
    ErrAPIReachFailed       = "Error while reaching the API"
    ErrDecodeResponseFailed = "Error while decoding the response"
    ErrNoPermissionView     = "User does not have permission to view Catalogs"
    ErrInvalidPuzzleIndex   = "Invalid puzzle index"
    ErrPuzzleOutOfBounds    = "Puzzle index out of bounds"
    ErrMissingRequiredFields = "Missing required fields in request"
)

// PuzzleResponse represents the API puzzle response
type PuzzleResponse struct {
    Author           string `json:"author"`
    Cipher           string `json:"cipher"`
    CompressedSize   int    `json:"compressedSize"`
    CreatedAt        string `json:"createdAt"`
    Difficulty       string `json:"difficulty"`
    ID               string `json:"id"`
    Language         string `json:"language"`
    Name             string `json:"name"`
    Title            string `json:"title"`
    Index            string `json:"index"`
    Obscure          string `json:"obscure"`
    UncompressedSize int    `json:"uncompressedSize"`
    HivecraftVersion string `json:"hivecraftVersion"`
    UpdatedAt        string `json:"updatedAt"`
}

// ThemeResponse represents the API theme response
type ThemeResponse struct {
    EnigmesCount int              `json:"enigmes_count"`
    Name         string           `json:"name"`
    Puzzles      []PuzzleResponse `json:"puzzles"`
    Size         int              `json:"size"`
}

// GetPuzzleInputRequest represents the request body for fetching puzzle input
type GetPuzzleInputRequest struct {
    CatalogID string `json:"catalogId" binding:"required"`
    ThemeName string `json:"themeName" binding:"required"`
    PuzzleID  string `json:"puzzleId" binding:"required"`
    SeedID    string `json:"userId" binding:"required"`
}