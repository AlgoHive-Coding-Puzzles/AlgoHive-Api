package middleware

import (
	"api/database"
	"api/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserFromRequest retrieves user from the context and handles error responses
// Returns the user or an error. If an error response has been sent, the caller can just return
func GetUserFromRequest(c *gin.Context) (models.User, error) {
    userID, exists := c.Get("userID")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return models.User{}, errors.New("user not authenticated")
    }

    var user models.User
    result := database.DB.Where("id = ?", userID).Preload("Roles").First(&user)
    if result.Error != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return models.User{}, result.Error
    }

    return user, nil
}