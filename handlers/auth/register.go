package auth

import (
	"api/database"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterUser handles registration of a new user
// @Summary User Register
// @Description Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "Registration Details"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /auth/register [post]
func RegisterUser(c *gin.Context) {
	var registerReq RegisterRequest
	
	if err := c.ShouldBindJSON(&registerReq); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	// Check if the email already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", registerReq.Email).First(&existingUser).Error; err == nil {
		respondWithError(c, http.StatusConflict, ErrEmailInUse)
		return
	}
	
	// Hash the password
	hashedPassword, err := utils.HashPassword(registerReq.Password)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrHashPasswordFailed)
		return
	}
	
	// Create a new user
	user := models.User{
		Email:     registerReq.Email,
		Password:  hashedPassword,
		Firstname: registerReq.Firstname,
		Lastname:  registerReq.Lastname,
		Blocked:   false,
	}
	
	if err := database.DB.Create(&user).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrUserCreateFailed)
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
	
	c.JSON(http.StatusCreated, AuthResponse{
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
