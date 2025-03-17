package catalogs

import (
	"github.com/gin-gonic/gin"
)

// Error message constants
const (
	ErrCatalogNotFound    = "API not found"
	ErrAPIReachFailed     = "Error while reaching the API"
	ErrDecodeResponseFailed = "Error while decoding the response"
	ErrNoPermissionView   = "User does not have permission to view Catalogs"
)

// PuzzleResponse represents the API puzzle response
type PuzzleResponse struct {
	Author          string `json:"author"`
	Cipher          string `json:"cipher"`
	CompressedSize  int    `json:"compressedSize"`
	CreatedAt       string `json:"createdAt"`
	Difficulty      string `json:"difficulty"`
	ID              string `json:"id"`
	Language        string `json:"language"`
	Name            string `json:"name"`
	Obscure         string `json:"obscure"`
	UncompressedSize int   `json:"uncompressedSize"`
	UpdatedAt       string `json:"updatedAt"`
}

// ThemeResponse represents the API theme response
type ThemeResponse struct {
	EnigmesCount int             `json:"enigmes_count"`
	Name         string          `json:"name"`
	Puzzles      []PuzzleResponse `json:"puzzles"`
	Size         int             `json:"size"`
}

// respondWithError sends a standardized error response
func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
