package roles

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AttachRoleToUser attaches a role to a user
// @Summary Attach a Role to a User
// @Description Attach a Role to a User
// @Tags Roles
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Router /roles/attach/{role_id}/to-user/{user_id} [post]
// @Security Bearer
func AttachRoleToUser(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        respondWithError(c, http.StatusUnauthorized, ErrNoPermissionAttach)
        return
    }

	// Get target user
	targetUserId := c.Param("user_id")
	var targetUser models.User
	
	if err := database.DB.Where("id = ?", targetUserId).Preload("Roles").First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrUserNotFound)
		return
	}

	// Get role to attach
	roleID := c.Param("role_id")
	
	var role models.Role
	if err := database.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}
	
	// Attach role to user
	targetUser.Roles = append(targetUser.Roles, &role)
	if err := database.DB.Save(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to attach role to user")
		return
	}
	
	c.JSON(http.StatusOK, targetUser)
}

// DetachRoleFromUser detaches a role from a user
// @Summary Detach a Role from a User
// @Description Detach a Role from a User
// @Tags Roles
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Router /roles/detach/{role_id}/from-user/{user_id} [delete]
// @Security Bearer
func DetachRoleFromUser(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDetach)
		return
	}

	// Get target user
	targetUserId := c.Param("user_id")
	var targetUser models.User
	
	if err := database.DB.Where("id = ?", targetUserId).Preload("Roles").First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrUserNotFound)
		return
	}
	
	// Get role to detach
	roleID := c.Param("role_id")

	var role models.Role
	if err := database.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}
	
	// Find and remove role from user
	for i, r := range targetUser.Roles {
		if r.ID == role.ID {
			targetUser.Roles = append(targetUser.Roles[:i], targetUser.Roles[i+1:]...)
			break
		}
	}
	
	if err := database.DB.Save(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to detach role from user")
		return
	}
	
	c.JSON(http.StatusOK, targetUser)
}
