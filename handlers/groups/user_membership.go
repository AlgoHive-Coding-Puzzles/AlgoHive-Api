package groups

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/response"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddUserToGroup adds a user to a group
// @Summary Add a user to a group
// @Description Add a user to a group, requires group management permissions
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param user_id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse "Group or user not found"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Failure 500 {object} response.ErrorResponse "Failed to add user to group"
// @Router /groups/{group_id}/users/{user_id} [post]
// @Security Bearer
func AddUserToGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")
    targetUserID := c.Param("user_id")
    
    // First check if the group exists and get basic info
    var group models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            First(&group, "id = ?", groupID).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
        return
    }

    // Check permissions - user must either own the groups or have management permissions
    if !UserOwnsTargetGroups(user.ID, groupID) && !userCanManageGroup(user.ID, &group) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionAddUser)
        return
    }

    // Verify the target user exists
    var targetUser models.User
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Select("id").
            First(&targetUser, "id = ?", targetUserID).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrUserNotFound)
        return
    }

    // Check if user is already a member of the group
    var membershipExists bool
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Table("user_groups").
            Select("COUNT(*) > 0").
            Where("user_id = ? AND group_id = ?", targetUserID, groupID).
            Find(&membershipExists).Error
    })

    // Only add user if they aren't already in the group
    if err != nil || membershipExists {
        // If there was an error or user already exists, return success
        // since the end state is what the caller wanted
        c.Status(http.StatusNoContent)
        return
    }

    // Add user to group as a transaction
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Model(&group).
            Association("Users").
            Append(&targetUser)
    })
    
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to add user to group")
        return
    }
    
    c.Status(http.StatusNoContent)
}

// RemoveUserFromGroup removes a user from a group
// @Summary Remove a user from a group
// @Description Remove a user from a group, requires group management permissions
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param user_id path string true "User ID" 
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse "Group or user not found"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Failure 500 {object} response.ErrorResponse "Failed to remove user from group"
// @Router /groups/{group_id}/users/{user_id} [delete]
// @Security Bearer
func RemoveUserFromGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")
    targetUserID := c.Param("user_id")
    
    // First check if the group exists
    var group models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            First(&group, "id = ?", groupID).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
        return
    }

    // Check permissions
    if !UserOwnsTargetGroups(user.ID, groupID) && !userCanManageGroup(user.ID, &group) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionRemoveUser)
        return
    }

    // Check if user exists
    var targetUser models.User
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Select("id").
            First(&targetUser, "id = ?", targetUserID).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrUserNotFound)
        return
    }

    // Check if membership exists before removal
    var membershipExists bool
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Table("user_groups").
            Select("COUNT(*) > 0").
            Where("user_id = ? AND group_id = ?", targetUserID, groupID).
            Find(&membershipExists).Error
    })

    // If user isn't in group, just return success since the desired state is already achieved
    if err != nil || !membershipExists {
        c.Status(http.StatusNoContent)
        return
    }

    // Remove the user from group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Model(&group).
            Association("Users").
            Delete(&targetUser)
    })
    
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to remove user from group")
        return
    }
    
    c.Status(http.StatusNoContent)
}