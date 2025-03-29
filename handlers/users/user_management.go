package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// createUser creates a new user with basic information
// firstName, lastName, email: basic user information
// returns: the created user and any error
func createUser(firstName, lastName, email string) (*models.User, error) {
    // Input validation
    if firstName == "" || lastName == "" || email == "" {
        return nil, errors.New("first name, last name, and email are required")
    }

    // Generate a default password
    hashedPassword, err := utils.CreateDefaultPassword()
    if err != nil {
        return nil, err
    }
    
    user := &models.User{
        Firstname:         firstName,
        Lastname:          lastName,
        Email:             email,
        Password:          hashedPassword,
        HasDefaultPassword: true,
    }
    
    return user, nil
}

// GetUsers retrieves all users accessible to the authenticated user
// @Summary Get All users that the current user has access to 
// @Description Get all users that the current user has access to from their roles -> scopes -> groups
// @Tags Users
// @Success 200 {array} models.User
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/ [get]
// @Security Bearer
func GetUsers(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    var users []models.User
    
    // Owners can see all users
    if permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        if err := database.DB.Preload("Roles").Preload("Groups").Find(&users).Error; err != nil {
            response.Error(c, http.StatusInternalServerError, ErrFailedToGetUsers)
            return
        }
    } else if len(user.Roles) == 0 {
        // For users without roles, only those in the same groups
        if err := getUsersInSameGroups(user.ID, &users); err != nil {
            response.Error(c, http.StatusInternalServerError, ErrFailedToGetUsers)
            return
        }
    } else {
        // For users with roles, use the role->scope->group hierarchy
        if err := getUsersFromRoleScopes(user.ID, &users); err != nil {
            response.Error(c, http.StatusInternalServerError, ErrFailedToGetUsers)
            return
        }
    }

    // Hide passwords in response
    for i := range users {
        users[i].Password = ""
    }

    c.JSON(http.StatusOK, users)
}

// getUsersInSameGroups retrieves all users who are in the same groups as the user
// userID: ID of the user
// users: pointer to the slice of users to fill
// returns: any error
func getUsersInSameGroups(userID string, users *[]models.User) error {
    err := database.DB.Raw(`
        SELECT DISTINCT u.*
        FROM users u
        JOIN user_groups ug ON u.id = ug.user_id
        JOIN groups g ON ug.group_id = g.id
        JOIN user_groups aug ON g.id = aug.group_id
        JOIN users au ON aug.user_id = au.id
        WHERE au.id = ?
    `, userID).Scan(users).Error
    
    if err != nil {
        return err
    }
    
    // Load associations if users were found
    if len(*users) > 0 {
        return database.DB.Preload("Roles").Preload("Groups").Where("id IN ?", pluckIDs(*users)).Find(users).Error
    }
    
    return nil
}

// getUsersFromRoleScopes retrieves all users accessible via roles->scopes->groups
// userID: ID of the user
// users: pointer to the slice of users to fill
// returns: any error
func getUsersFromRoleScopes(userID string, users *[]models.User) error {
    var userIDs []string
    if err := database.DB.Raw(`
        SELECT DISTINCT u.id
        FROM users u
        JOIN user_groups ug ON u.id = ug.user_id
        JOIN groups g ON ug.group_id = g.id
        JOIN scopes s ON g.scope_id = s.id
        JOIN role_scopes rs ON s.id = rs.scope_id
        JOIN roles r ON rs.role_id = r.id
        JOIN user_roles ur ON r.id = ur.role_id
        WHERE ur.user_id = ?
    `, userID).Pluck("id", &userIDs).Error; err != nil {
        return err
    }
    
    // If no users found, return empty slice
    if len(userIDs) == 0 {
        *users = []models.User{}
        return nil
    }
    
    // Retrieve users with their associations
    return database.DB.Preload("Roles").Preload("Groups").Where("id IN ?", userIDs).Find(users).Error
}

// pluckIDs extracts the IDs from a slice of users
func pluckIDs(users []models.User) []string {
    ids := make([]string, len(users))
    for i, user := range users {
        ids[i] = user.ID
    }
    return ids
}

// DeleteUser deletes a user by ID
// @Summary Delete User
// @Description Delete a user by ID, if user is Staff, requires ownership permission
// @Tags Users
// @Param id path string true "User ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/{id} [delete]
// @Security Bearer
func DeleteUser(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    userID := c.Param("id")
    if userID == "" {
        response.Error(c, http.StatusBadRequest, "User ID is required")
        return
    }

    // Check if target user exists
    var targetUser models.User
    if err := database.DB.Where("id = ?", userID).First(&targetUser).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            response.Error(c, http.StatusNotFound, ErrUserNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Database error when finding user")
        }
        return
    }

    // Check permissions
    if !HasPermissionForUser(user, targetUser.ID, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, ErrNoPermissionDelete)
        return
    }

    // Start a transaction to ensure atomicity of operations
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Delete associations first
    if err := tx.Model(&targetUser).Association("Roles").Clear(); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedAssociationRoles)
        return
    }
    
    if err := tx.Model(&targetUser).Association("Groups").Clear(); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedAssociationGroups)
        return
    }

    // Delete the user
    if err := tx.Delete(&targetUser).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedToDeleteUser)
        return
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    c.Status(http.StatusNoContent)
}

// BulkDeleteUsers deletes multiple users by IDs
// @Summary Bulk Delete Users
// @Description Bulk delete users by IDs
// @Tags Users
// @Accept json
// @Produce json
// @Param ids body []string true "User IDs"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/bulk [delete]
// @Security Bearer
func BulkDeleteUsers(c *gin.Context) {
    // Get authenticated user
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Parse user IDs from request body
    var userIDs []string
    if err := c.ShouldBindJSON(&userIDs); err != nil {
        response.Error(c, http.StatusBadRequest, ErrInvalidUserIDs)
        return
    }
    
    if len(userIDs) == 0 {
        response.Error(c, http.StatusBadRequest, ErrEmptyUserIDs)
        return
    }

    // Check if users exist
    var users []models.User
    if err := database.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedToGetUsers)
        return
    }
    
    if len(users) == 0 {
        response.Error(c, http.StatusNotFound, ErrUserNotFound)
        return
    }
    
    // Check permissions - must have permission for all users
    for _, targetUser := range users {
        if !HasPermissionForUser(user, targetUser.ID, permissions.OWNER) {
            response.Error(c, http.StatusForbidden, ErrNoPermissionDelete)
            return
        }
    }
    
    // Start a transaction to ensure atomicity of operations
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Delete associations first
    for _, targetUser := range users {
        if err := tx.Model(&targetUser).Association("Roles").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAssociationRoles)
            return
        }
        
        if err := tx.Model(&targetUser).Association("Groups").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAssociationGroups)
            return
        }
    }
    
    // Delete the users
    if err := tx.Where("id IN ?", userIDs).Delete(&models.User{}).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedToDeleteUsers)
        return
    }
    
    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    c.Status(http.StatusNoContent)
}

// ToggleBlockUser toggles the block status of a user
// @Summary Toggle Block User
// @Description Toggle the block status of a user
// @Tags Users
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/block/{id} [put]
// @Security Bearer
func ToggleBlockUser(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    userID := c.Param("id")
    if userID == "" {
        response.Error(c, http.StatusBadRequest, "User ID is required")
        return
    }
    
    // Check if target user exists
    var targetUser models.User
    if err := database.DB.Where("id = ?", userID).First(&targetUser).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            response.Error(c, http.StatusNotFound, ErrUserNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Database error when finding user")
        }
        return
    }

    // Check permissions
    if !HasPermissionForUser(user, targetUser.ID, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, ErrNoPermissionBlock)
        return
    }

    // Use transaction for update
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Toggle block status
    updateFields := map[string]interface{}{
        "blocked": !targetUser.Blocked,
    }
    
    if err := tx.Model(&targetUser).Updates(updateFields).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update user block status")
        return
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    // Retrieve the updated user to return
    if err := database.DB.Preload("Roles").Preload("Groups").Where("id = ?", userID).First(&targetUser).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "User updated but failed to retrieve updated data")
        return
    }
    
    // Hide password
    targetUser.Password = ""
    
    c.JSON(http.StatusOK, targetUser)
}