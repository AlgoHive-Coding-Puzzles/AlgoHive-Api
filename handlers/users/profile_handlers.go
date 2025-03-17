package users

import (
	"api/config"
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserProfile retrieves the authenticated user's profile
// @Summary Get User Profile
// @Description Get the profile information of the authenticated user
// @Tags Users
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Router /user/profile [get]
// @Security Bearer
func GetUserProfile(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
    
    // Hide password from response for security
    user.Password = ""
    
    c.JSON(http.StatusOK, user)
}

// UpdateUserProfile updates the authenticated user's profile
// @Summary Update User Profile
// @Description Update the profile information of the authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "User Profile"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/profile [put]
// @Security Bearer
func UpdateUserProfile(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	var userUpdate models.User
	if err := c.ShouldBindJSON(&userUpdate); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	// Update only allowed fields
	user.Email = userUpdate.Email
	user.Firstname = userUpdate.Firstname
	user.Lastname = userUpdate.Lastname
	
	if err := database.DB.Save(&user).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	
	c.JSON(http.StatusOK, user)
}

// UpdateTargetUserProfile updates the target user's profile
// @Summary Update Target User Profile
// @Description Update the profile information of the target user
// @Tags Users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param user body models.User true "User Profile"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/{id} [put]
// @Security Bearer
func UpdateTargetUserProfile(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	// Get the target user ID from the URL parameter
	userId := c.Param("id")
	if userId == "" {
		respondWithError(c, http.StatusBadRequest, "User ID is required")
		return
	}
	
	// Find the target user by ID
	var userUpdate models.User
	if err := database.DB.Where("id = ?", userId).First(&userUpdate).Error; err != nil {
		respondWithError(c, http.StatusNotFound, "User not found")
		return
	}

	// Check if the authenticated user has permission to update the target user's profile
	if !HasPermissionForUser(user, userUpdate.ID, permissions.GROUPS) && !HasPermissionForUser(user, userUpdate.ID, permissions.OWNER) {
		respondWithError(c, http.StatusForbidden, "You do not have permission to update this user's profile")
		return
	}
	
	if err := c.ShouldBindJSON(&userUpdate); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	userUpdate.ID = userId
	
	if err := database.DB.Save(&userUpdate).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	
	c.JSON(http.StatusOK, userUpdate)
}

// ResetUserPassword resets the target user's password
// @Summary Reset Target User Password
// @Description Reset the password of the target user
// @Tags Users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/resetpass/{id} [put]
// @Security Bearer
func ResetUserPassword(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	// Get the target user ID from the URL parameter
	userId := c.Param("id")
	if userId == "" {
		respondWithError(c, http.StatusBadRequest, "User ID is required")
		return
	}
	// Find the target user by ID
	var userUpdate models.User
	if err := database.DB.Where("id = ?", userId).First(&userUpdate).Error; err != nil {
		respondWithError(c, http.StatusNotFound, "User not found")
		return
	}

	// Check if the authenticated user has permission to reset the target user's password
	if !HasPermissionForUser(user, userUpdate.ID, permissions.GROUPS) {
		respondWithError(c, http.StatusForbidden, "You do not have permission to reset this user's password")
		return
	}
	
	// Create a default admin user with a default hashed password from either the .env file or the DefaultPassword constant
	password := config.DefaultPassword
	if config.DefaultPassword != "" {
		password = config.DefaultPassword
	}

	password, err = utils.HashPassword(password)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	userUpdate.Password = password
	
	if err := database.DB.Save(&userUpdate).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	
	c.JSON(http.StatusOK, user)
}

// UpdateUserPassword updates the current user's password
// @Summary Update User Password
// @Description Update the password of the current user
// @Tags Users
// @Accept json
// @Produce json
// @Param passwords body PasswordUpdate true "Password Update"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/profile/password [put]
// @Security Bearer
func UpdateUserPassword(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	var passwordUpdate PasswordUpdate
	if err := c.ShouldBindJSON(&passwordUpdate); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	if !utils.CheckPasswordHash(passwordUpdate.OldPassword, user.Password) {
		respondWithError(c, http.StatusUnauthorized, "Old password is incorrect")
		return
	}
	
	hashedPassword, err := utils.HashPassword(passwordUpdate.NewPassword)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to hash new password")
		return
	}
	
	user.Password = hashedPassword
	
	if err := database.DB.Save(&user).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update password")
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}
