package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// createUser creates a new user with basic information
// firstName, lastName, email: basic user information
// returns: the created user and any error
func createUser(firstName, lastName, email string) (*models.User, error) {
	var user models.User
	user.Firstname = firstName
	user.Lastname = lastName
	user.Email = email
	
	// Generate a default password
	hashedPassword, err := utils.CreateDefaultPassword()
	if err != nil {
		return nil, err
	}
	user.Password = hashedPassword
	
	// Create the user
	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	
	return &user, nil
}

// GetUsers retrieves all users accessible to the authenticated user
// @Summary Get All users that the curren user has access to 
// @Description Get all users that the current user has access to from his roles -> scopes -> groups
// @Tags Users
// @Success 200 {object} []models.User
// @Router /user/ [get]
// @Security Bearer
func GetUsers(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	var users []models.User
	
	// Owners can see all users
	if permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		if err := database.DB.Preload("Roles").Preload("Groups").Find(&users).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, ErrFailedToGetUsers)
			return
		}
	} else {
			// For users without roles, only those in the same groups
		if len(user.Roles) == 0 {
			if err := getUsersInSameGroups(user.ID, &users); err != nil {
				respondWithError(c, http.StatusInternalServerError, ErrFailedToGetUsers)
				return
			}
		} else {
				// For users with roles, use the role->scope->group hierarchy
			if err := getUsersFromRoleScopes(user.ID, &users); err != nil {
				respondWithError(c, http.StatusInternalServerError, ErrFailedToGetUsers)
				return
			}
		}
	}

	c.JSON(http.StatusOK, users)
}

// getUsersInSameGroups retrieves all users who are in the same groups as the user
// userID: ID of the user
// users: pointer to the slice of users to fill
// returns: any error
func getUsersInSameGroups(userID string, users *[]models.User) error {
	return database.DB.Raw(`
		SELECT DISTINCT u.*
			FROM users u
			JOIN user_groups ug ON u.id = ug.user_id
			JOIN groups g ON ug.group_id = g.id
			JOIN user_groups aug ON g.id = aug.group_id
			JOIN users au ON aug.user_id = au.id
			WHERE au.id = ?
	`, userID).Scan(users).Error
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

// DeleteUser deletes a user by ID
// @Summary Delete User
// @Description Delete a user by ID, if user isStaff, required ownership permission
// @Tags Users
// @Param id path string true "User ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/{id} [delete]
// @Security Bearer
func DeleteUser(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    userID := c.Param("id")
    var targetUser models.User
    
    // Check if target user exists
    if err := database.DB.Where("id = ?", userID).First(&targetUser).Error; err != nil {
        respondWithError(c, http.StatusNotFound, ErrNotFound)
        return
    }

    // Check permissions
    if !HasPermissionForUser(user, targetUser.ID, permissions.OWNER) {
        respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

    // Start a transaction to ensure atomicity of operations
    tx := database.DB.Begin()

    // Delete associations first
    if err := tx.Model(&targetUser).Association("Roles").Clear(); err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedAssociationRoles)
        return
    }
    
    if err := tx.Model(&targetUser).Association("Groups").Clear(); err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedAssociationGroups)
        return
    }

    // Delete the user
    if err := tx.Delete(&targetUser).Error; err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedToDeleteUser)
        return
    }

    // Commit the transaction
    tx.Commit()
    
    c.Status(http.StatusNoContent)
}

// ToggleBlockUser toggles the block status of a user
// @Summary Toggle block user
// @Description Toggle the block status of a user
// @Tags Users
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/block/{id} [put]
// @Security Bearer
func ToggleBlockUser(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	userID := c.Param("id")
	var targetUser models.User
	
	// Check if target user exists
	if err := database.DB.Where("id = ?", userID).First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	// Check permissions
	if !HasPermissionForUser(user, targetUser.ID, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionBlock)
		return
	}

	// Toggle block status
	targetUser.Blocked = !targetUser.Blocked
	if err := database.DB.Save(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	c.JSON(http.StatusOK, targetUser)
}
