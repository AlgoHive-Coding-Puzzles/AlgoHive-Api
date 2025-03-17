package auth

import (
	"api/database"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Login handles user authentication and returns a JWT token
// @Summary User Login
// @Description Authenticate a user and return a JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login Credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var loginReq LoginRequest
	
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	var user models.User
	if err := database.DB.Where("email = ?", loginReq.Email).Preload("Roles").Preload("Groups").First(&user).Error; err != nil {
		respondWithError(c, http.StatusUnauthorized, ErrInvalidCredentials)
		return
	}
	
	// Check if the user is blocked
	if user.Blocked {
		respondWithError(c, http.StatusUnauthorized, ErrAccountBlocked)
		return
	}
	
	// Verify the password
	if !utils.CheckPasswordHash(loginReq.Password, user.Password) {
		respondWithError(c, http.StatusUnauthorized, ErrInvalidCredentials)
		return
	}
	
	// Generate a JWT token
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrTokenGenerateFailed)
		return
	}
	
	// Set the token as an HTTP-only cookie
	setCookieToken(c, token)
	
	// Update the last connection time
	now := time.Now()
	user.LastConnected = &now
	if err := database.DB.Save(&user).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrTokenGenerateFailed)
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:         token,
		UserID:        user.ID,
		Email:         user.Email,
		Firstname:     user.Firstname,
		Lastname:      user.Lastname,
		LastConnected: user.LastConnected,
		Permissions:   permissions.MergeRolePermissions(user.Roles),
		Roles:         utils.ConvertRoles(user.Roles),
		Groups:        utils.ConvertGroups(user.Groups),
	})
}
