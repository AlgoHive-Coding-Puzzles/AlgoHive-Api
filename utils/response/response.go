package response

import (
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
    Message string `json:"message"`
    Code    int    `json:"code"`
}

// Error sends a standardized error response
func Error(c *gin.Context, status int, message string) {
    c.JSON(status, gin.H{"error": message})
}

// Success sends a standardized success response
func Success(c *gin.Context, status int, data interface{}) {
    c.JSON(status, gin.H{"data": data})
}

// ValidationError sends a response for validation errors
func ValidationError(c *gin.Context, errors map[string]string) {
    c.JSON(400, gin.H{"errors": errors})
}