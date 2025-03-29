package roles

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"api/utils/response"
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
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /roles/attach/{role_id}/to-user/{user_id} [post]
// @Security Bearer
func AttachRoleToUser(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionAttach)
        return
    }

    // Get role and user IDs
    targetUserID := c.Param("user_id")
    roleID := c.Param("role_id")

    // Start a transaction
    tx := database.DB.Begin()

    // Get target user
    var targetUser models.User
    if err := tx.Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrUserNotFound)
        return
    }

    // Check if role exists
    var role models.Role
    if err := tx.Where("id = ?", roleID).First(&role).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    // Check if user already has this role
    var count int64
    if err := tx.Model(&models.User{}).
        Joins("JOIN user_roles ON users.id = user_roles.user_id").
        Where("users.id = ? AND user_roles.role_id = ?", targetUserID, roleID).
        Count(&count).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to check existing roles")
        return
    }

    // Only attach if not already attached
    if count == 0 {
        if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", targetUserID, roleID).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to attach role to user")
            return
        }
    } else {
        tx.Rollback()
        response.Error(c, http.StatusConflict, "User already has this role")
        return
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Fetch the updated user with roles for response
    if err := database.DB.Preload("Roles").Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to fetch updated user data")
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
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /roles/detach/{role_id}/from-user/{user_id} [delete]
// @Security Bearer
func DetachRoleFromUser(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionDetach)
        return
    }

    // Get role and user IDs
    targetUserID := c.Param("user_id")
    roleID := c.Param("role_id")

    // Start a transaction
    tx := database.DB.Begin()

    // Verify user exists
    var targetUser models.User
    if err := tx.Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrUserNotFound)
        return
    }

    // Verify role exists
    var role models.Role
    if err := tx.Where("id = ?", roleID).First(&role).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    // Check if user has this role
    var count int64
    if err := tx.Model(&models.User{}).
        Joins("JOIN user_roles ON users.id = user_roles.user_id").
        Where("users.id = ? AND user_roles.role_id = ?", targetUserID, roleID).
        Count(&count).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to check existing roles")
        return
    }

    if count == 0 {
        tx.Rollback()
        response.Error(c, http.StatusBadRequest, "User does not have this role")
        return
    }

    // Detach the role directly from the join table
    if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ? AND role_id = ?", targetUserID, roleID).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to detach role from user")
        return
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Fetch the updated user with roles for response
    if err := database.DB.Preload("Roles").Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to fetch updated user data")
        return
    }

    c.JSON(http.StatusOK, targetUser)
}