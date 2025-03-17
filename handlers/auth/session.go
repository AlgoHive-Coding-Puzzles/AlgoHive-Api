package auth

import (
	"api/database"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// getTokenFromRequest retrieves the token from the cookie or Authorization header
func getTokenFromRequest(c *gin.Context) (string, error) {
	// Retrieve token from the cookie first
	token, err := c.Cookie("auth_token")

	// If no cookie, try the Authorization header
	if err != nil {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			return "", http.ErrNoCookie
		}

		// Extract the actual token from "Bearer token"
		token = authHeader[7:] // Remove "Bearer " prefix
	}

	return token, nil
}

// Logout handles user logout by invalidating their token
// @Summary User Logout
// @Description Logout a user by invalidating their token
// @Tags Auth
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/logout [post]
// @Security Bearer
func Logout(c *gin.Context) {
	// Retrieve the token
	token, err := getTokenFromRequest(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, ErrNoTokenProvided)
		return
	}

	// Validate the token to obtain the expiration time
	claims, err := utils.ValidateToken(token)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, ErrInvalidToken)
		return
	}

	// Calculate the remaining time until token expiration
	expirationTime := claims.ExpiresAt.Time
	remainingTime := time.Until(expirationTime)

	// Add to Redis blacklist
	ctx := context.Background()
	redisKey := "token:blacklist:" + token
	err = database.REDIS.Set(ctx, redisKey, "1", remainingTime).Err()
	if err != nil {
		
		respondWithError(c, http.StatusInternalServerError, ErrLogoutFailed)
		return
	}

	// Clear the authentication cookie
	c.SetCookie("auth_token", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": ErrLogoutSuccess})
}

// CheckAuth checks if the session token is still valid and returns user data
// @Summary Check if the sent cookie token session is still valid and return user data
// @Description Check if the sent cookie token session is still valid and return user data
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} AuthResponse
// @Failure 401 {object} map[string]string
// @Router /auth/check [get]
// @Security Bearer
func CheckAuth(c *gin.Context) {
	// Retrieve the token
	token, err := getTokenFromRequest(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, ErrNoTokenProvided)
		return
	}

	// Validate the token
	claims, err := utils.ValidateToken(token)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, ErrInvalidExpiredToken)
		return
	}

	// Retrieve user data
	var user models.User
	if err := database.DB.Where("id = ?", claims.UserID).Preload("Roles").Preload("Groups").First(&user).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrUserNotFound)
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		UserID:        user.ID,
		Email:         user.Email,
		Firstname:     user.Firstname,
		Lastname:      user.Lastname,
		LastConnected: user.LastConnected,
		Blocked:       user.Blocked,
		Permissions:   permissions.MergeRolePermissions(user.Roles),
		Roles:         utils.ConvertRoles(user.Roles),
		Groups:        utils.ConvertGroups(user.Groups),
	})
}
