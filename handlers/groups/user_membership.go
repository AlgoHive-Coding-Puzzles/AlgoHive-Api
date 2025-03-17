package groups

import (
	"api/database"
	"api/middleware"
	"api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddUserToGroup ajoute un utilisateur à un groupe
// @Summary Add a user to a group
// @Description Add a user to a group
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param user_id path string true "User ID"
// @Success 204 {object} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /groups/{group_id}/users/{user_id} [post]
// @Security Bearer
func AddUserToGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")
	if !UserOwnsTargetGroups(user.ID, groupID) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionAddUser)
		return
	}

	var group models.Group
	if err := database.DB.Where("id = ?", groupID).Preload("Users").First(&group).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
		return
	}

	if !userCanManageGroup(user.ID, &group) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionAddUser)
		return
	}

	targetUserID := c.Param("user_id")
	var targetUser models.User
	if err := database.DB.Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrUserNotFound)
		return
	}

	// Ajouter l'utilisateur au groupe
	if err := database.DB.Model(&group).Association("Users").Append(&targetUser); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to add user to group")
		return
	}
	
	c.Status(http.StatusNoContent)
}

// RemoveUserFromGroup retire un utilisateur d'un groupe
// @Summary Remove a user from a group
// @Description Remove a user from a group
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param user_id path string true "User ID"
// @Success 204 {object} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /groups/{group_id}/users/{user_id} [delete]
// @Security Bearer
func RemoveUserFromGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")
	if !UserOwnsTargetGroups(user.ID, groupID) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionRemoveUser)
		return
	}

	var group models.Group
	if err := database.DB.Where("id = ?", groupID).Preload("Users").First(&group).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
		return
	}

	if !userCanManageGroup(user.ID, &group) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionRemoveUser)
		return
	}

	targetUserID := c.Param("user_id")
	var targetUser models.User
	if err := database.DB.Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrUserNotFound)
		return
	}

	// Retirer l'utilisateur du groupe
	if err := database.DB.Model(&group).Association("Users").Delete(&targetUser); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to remove user from group")
		return
	}
	
	c.Status(http.StatusNoContent)
}
