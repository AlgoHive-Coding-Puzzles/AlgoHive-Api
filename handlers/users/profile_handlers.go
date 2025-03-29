package users

import (
	"api/config"
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserProfile retrieves the authenticated user's profile
// @Summary Get User Profile
// @Description Get the profile information of the authenticated user
// @Tags Users
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Router /user/profile [get]
// @Security Bearer
func GetUserProfile(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
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
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/profile [put]
// @Security Bearer
func UpdateUserProfile(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }
    
    var userUpdate models.User
    if err := c.ShouldBindJSON(&userUpdate); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    
    // Input validation
    if userUpdate.Email == "" || userUpdate.Firstname == "" || userUpdate.Lastname == "" {
        response.Error(c, http.StatusBadRequest, "Email, first name, and last name are required")
        return
    }
    
    // Use a transaction to ensure atomicity
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Update only allowed fields
    updatedFields := map[string]interface{}{
        "email":     userUpdate.Email,
        "firstname": userUpdate.Firstname,
        "lastname":  userUpdate.Lastname,
    }
    
    if err := tx.Model(&user).Updates(updatedFields).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update profile")
        return
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    // Retrieve the updated user to return
    var updatedUser models.User
    if err := database.DB.First(&updatedUser, "id = ?", user.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Profile updated but failed to retrieve updated data")
        return
    }
    
    // Hide password
    updatedUser.Password = ""

    // Invalidate the cache for the user session
    cacheKey := UserCacheKeyPrefix + user.ID
    if err := database.REDIS.Del(c, cacheKey).Err(); err != nil {
        log.Printf("Failed to invalidate user session cache: %v", err)
    }
    
    c.JSON(http.StatusOK, updatedUser)
}

// UpdateTargetUserProfile updates the target user's profile
// @Summary Update Target User Profile
// @Description Update the profile information of the target user
// @Tags Users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param user body UserProfileUpdate true "User Profile Update with optional roles and groups"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/{id} [put]
// @Security Bearer
func UpdateTargetUserProfile(c *gin.Context) {
    currentUser, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }
    
    // Get the target user ID from the URL parameter
    userId := c.Param("id")
    if userId == "" {
        response.Error(c, http.StatusBadRequest, "User ID is required")
        return
    }
    
    // Find the target user by ID
    var targetUser models.User
    if err := database.DB.First(&targetUser, "id = ?", userId).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrUserNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Database error when finding user")
        }
        return
    }

    // Check if the authenticated user has permission to update the target user's profile
    if !HasPermissionForUser(currentUser, targetUser.ID, permissions.GROUPS) && 
       !HasPermissionForUser(currentUser, targetUser.ID, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, "You do not have permission to update this user's profile")
        return
    }
    
    var profileUpdate UserProfileUpdate
    if err := c.ShouldBindJSON(&profileUpdate); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    
    userUpdate := profileUpdate.User
    
    // Input validation
    if userUpdate.Email == "" || userUpdate.Firstname == "" || userUpdate.Lastname == "" {
        response.Error(c, http.StatusBadRequest, "Email, first name, and last name are required")
        return
    }
    
    // Use a transaction to ensure atomicity
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Update only allowed fields
    updatedFields := map[string]interface{}{
        "email":     userUpdate.Email,
        "firstname": userUpdate.Firstname,
        "lastname":  userUpdate.Lastname,
    }
    
    if err := tx.Model(&targetUser).Where("id = ?", userId).Updates(updatedFields).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update profile")
        return
    }
    
    // Update roles if provided
    if len(profileUpdate.RoleIDs) > 0 {
        // Check if the provided role IDs exist
        var roles []*models.Role
        if err := tx.Where("id IN ?", profileUpdate.RoleIDs).Find(&roles).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to find roles")
            return
        }
        
        // Check if all provided role IDs exist
        if len(roles) != len(profileUpdate.RoleIDs) {
            tx.Rollback()
            response.Error(c, http.StatusBadRequest, ErrRoleNotFound)
            return
        }
        
        // First, clear existing associations
        if err := tx.Model(&targetUser).Association("Roles").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAssociationRoles)
            return
        }
        
        // Set new associations
        if err := tx.Model(&targetUser).Association("Roles").Replace(roles); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to update user roles")
            return
        }
    }
    
    // Update groups if provided
    if len(profileUpdate.GroupIDs) > 0 {
        // Check if the provided group IDs exist
        var groups []*models.Group
        if err := tx.Where("id IN ?", profileUpdate.GroupIDs).Find(&groups).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to find groups")
            return
        }
        
        // Check if all provided group IDs exist
        if len(groups) != len(profileUpdate.GroupIDs) {
            tx.Rollback()
            response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
            return
        }
        
        // First, clear existing associations
        if err := tx.Model(&targetUser).Association("Groups").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAssociationGroups)
            return
        }
        
        // Set new associations
        if err := tx.Model(&targetUser).Association("Groups").Replace(groups); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to update user groups")
            return
        }
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    // Retrieve the updated user to return with associations
    if err := database.DB.Preload("Roles").Preload("Groups").Where("id = ?", userId).First(&targetUser).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Profile updated but failed to retrieve updated data")
        return
    }
    
    // Hide password
    targetUser.Password = ""
    
    c.JSON(http.StatusOK, targetUser)
}

// ResetUserPassword resets the target user's password
// @Summary Reset Target User Password
// @Description Reset the password of the target user
// @Tags Users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/resetpass/{id} [put]
// @Security Bearer
func ResetUserPassword(c *gin.Context) {
    currentUser, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }
    
    // Get the target user ID from the URL parameter
    userId := c.Param("id")
    if userId == "" {
        response.Error(c, http.StatusBadRequest, "User ID is required")
        return
    }
    
    // Find the target user by ID
    var targetUser models.User
    if err := database.DB.Where("id = ?", userId).First(&targetUser).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrUserNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Database error when finding user")
        }
        return
    }

    // Check if the authenticated user has permission to reset the target user's password
    if !HasPermissionForUser(currentUser, targetUser.ID, permissions.GROUPS) && 
       !permissions.RolesHavePermission(currentUser.Roles, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, "You do not have permission to reset this user's password")
        return
    }
    
    // Get default password from config or use a fallback
    password := config.DefaultPassword
    if password == "" {
        response.Error(c, http.StatusInternalServerError, "Default password not configured")
        return
    }

    hashedPassword, err := utils.HashPassword(password)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedToHashPassword)
        return
    }

    // Use a transaction to ensure atomicity
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Update password fields
    updateFields := map[string]interface{}{
        "password":            hashedPassword,
        "has_default_password": true,
    }
    
    if err := tx.Model(&targetUser).Updates(updateFields).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to reset password")
        return
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Password has been reset successfully",
        "user_id": userId,
    })
}

// UpdateUserPassword updates the current user's password
// @Summary Update User Password
// @Description Update the password of the current user
// @Tags Users
// @Accept json
// @Produce json
// @Param passwords body PasswordUpdate true "Password Update"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/profile/password [put]
// @Security Bearer
func UpdateUserPassword(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }
    
    var passwordUpdate PasswordUpdate
    if err := c.ShouldBindJSON(&passwordUpdate); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    
    // Input validation
    if passwordUpdate.CurrentPassword == "" || passwordUpdate.NewPassword == "" {
        response.Error(c, http.StatusBadRequest, "Current password and new password are required")
        return
    }
    
    if !utils.CheckPasswordHash(passwordUpdate.CurrentPassword, user.Password) {
        response.Error(c, http.StatusUnauthorized, "Current password is incorrect")
        return
    }
    
    // Validate password strength
    if len(passwordUpdate.NewPassword) < 8 {
        response.Error(c, http.StatusBadRequest, "New password must be at least 8 characters long")
        return
    }
    
    hashedPassword, err := utils.HashPassword(passwordUpdate.NewPassword)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedToHashPassword)
        return
    }
    
    // Use a transaction to ensure atomicity
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Update password fields
    updateFields := map[string]interface{}{
        "password":             hashedPassword,
        "has_default_password": false,
    }
    
    if err := tx.Model(&user).Updates(updateFields).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update password")
        return
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}