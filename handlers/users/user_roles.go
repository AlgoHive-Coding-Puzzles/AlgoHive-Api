package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateUserAndAttachRoles creates a user and attaches roles to it
// @Summary Create a user and attach one or more roles
// @Description Create a new user and attach one or more roles to it
// @Tags Users
// @Accept json
// @Produce json
// @Param UserWithRoles body UserWithRoles true "User Profile with Roles"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Router /user/roles [post]
// @Security Bearer
func CreateUserAndAttachRoles(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionRoles)
		return
	}

	var userWithRoles UserWithRoles
	if err := c.ShouldBindJSON(&userWithRoles); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check that the roles exist
	var roles []models.Role
	if err := database.DB.Where("id IN (?)", userWithRoles.Roles).Find(&roles).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}
	
	if len(roles) == 0 {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}

	// Create the user
	targetUser, err := createUser(userWithRoles.FirstName, userWithRoles.LastName, userWithRoles.Email)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedToHashPassword)
		return
	}

	// Attach roles to the user
	for i := range roles {
		if err := database.DB.Model(targetUser).Association("Roles").Append(&roles[i]); err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to attach role to user")
			return
		}
	}

	c.JSON(http.StatusCreated, targetUser)
}

// GetUsersFromRoles retrieves all users accessible via specific roles
// @Summary Get All users that the given roles have access to
// @Description Get all users that the given roles have access to from their roles -> scopes -> groups -> users
// @Tags Users
// @Param roles query []string true "Roles IDs"
// @Success 200 {object} []models.User
// @Router /user/roles [get]
// @Security Bearer
func GetUsersFromRoles(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.IsStaff(user) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUsersRoles)
		return
	}

    // Retrieve role IDs from query parameters
    rolesParam := c.QueryArray("roles")
    
    // If a single comma-separated string is received, split it
    var roles []string
    if len(rolesParam) == 1 && strings.Contains(rolesParam[0], ",") {
        roles = strings.Split(rolesParam[0], ",")
    } else {
        roles = rolesParam
    }

    // Ensure that at least one role is provided
    if len(roles) == 0 {
        respondWithError(c, http.StatusBadRequest, ErrRolesRequired)
        return
    }

    // Retrieve users accessible via these roles
	users, err := getUsersFromRoleIDs(roles)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedToGetUsers)
		return
	}

	c.JSON(http.StatusOK, users)
}

// getUsersFromRoleIDs retrieves all users accessible via the specified roles
// roleIDs: IDs of the roles
// returns: the list of users and any error
func getUsersFromRoleIDs(roleIDs []string) ([]models.User, error) {
	var userIDs []string
	if err := database.DB.Raw(`
		SELECT DISTINCT u.id
			FROM users u
			JOIN user_groups ug ON u.id = ug.user_id
			JOIN groups g ON ug.group_id = g.id
			JOIN scopes s ON g.scope_id = s.id
			JOIN role_scopes rs ON s.id = rs.scope_id
			JOIN roles r ON rs.role_id = r.id
			WHERE r.id IN ?
	`, roleIDs).Pluck("id", &userIDs).Error; err != nil {
		return nil, err
	}

	var users []models.User
	if len(userIDs) > 0 {
		if err := database.DB.Preload("Roles").Preload("Groups").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	
	return users, nil
}

// UpdateUserRoles updates a user's roles
// @Summary Update the roles of a user
// @Description Update the roles of a user
// @Tags Users
// @Accept json
// @Produce json
// @Param roles body UserIdWithRoles true "User ID with Roles"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/roles/{userId} [put]
// @Security Bearer
func UpdateUserRoles(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.IsStaff(user) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUsersRoles)
		return
	}

	var userIdWithRoles UserIdWithRoles
	if err := c.ShouldBindJSON(&userIdWithRoles); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	var targetUser models.User
	if err := database.DB.Where("id = ?", userIdWithRoles.UserId).Preload("Roles").First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrUserNotFound)
		return
	}
	// Check that the roles exist
	var roles []models.Role
	if err := database.DB.Where("id IN (?)", userIdWithRoles.Roles).Find(&roles).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}
	if len(roles) == 0 {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}
	// Remove existing associations
	if err := database.DB.Model(&targetUser).Association("Roles").Clear(); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to clear user roles")
		return
	}
	// Attach new roles to the user
	for i := range roles {
		if err := database.DB.Model(&targetUser).Association("Roles").Append(&roles[i]); err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to attach role to user")
			return
		}
	}
	c.JSON(http.StatusOK, targetUser)
}