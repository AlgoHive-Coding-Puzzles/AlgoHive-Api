package groups

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Constants for database operations
const (
    defaultQueryTimeout = 5 * time.Second
)

// withTimeout executes a database operation with a timeout context
// dbOperation: The database operation function to execute with timeout
// returns: Error if the operation fails or times out
func withTimeout(dbOperation func(context.Context) error) error {
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    return dbOperation(ctx)
}

// transactionHandler creates and manages a database transaction
// c: The gin context for the HTTP request
// txFunc: The function to execute within the transaction
func transactionHandler(c *gin.Context, txFunc func(*gorm.DB) error) {
    tx := database.DB.Begin()
    
    if err := txFunc(tx); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    
    tx.Commit()
}

// GetAllGroups retrieves all groups
// @Summary Get all groups
// @Description Get all groups, only accessible to users with the GROUPS permission
// @Tags Groups
// @Accept json
// @Produce json
// @Success 200 {array} models.Group
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /groups [get]
// @Security Bearer
func GetAllGroups(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.GROUPS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    var groups []models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).Find(&groups).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFetchingGroups)
        return
    }
    
    c.JSON(http.StatusOK, groups)
}

// GetGroup retrieves a group by ID
// @Summary Get a group
// @Description Get a group with its users and competitions
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Success 200 {object} models.Group
// @Failure 400 {object} response.ErrorResponse "Group not found"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Router /groups/{group_id} [get]
// @Security Bearer
func GetGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")
    var group models.Group
    
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Where("id = ?", groupID).
            Preload("Users").
            Preload("Competitions").
            First(&group).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
        return
    }

    if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionViewGroup)
        return
    }

    c.JSON(http.StatusOK, group)
}

// CreateGroup creates a new group
// @Summary Create a group
// @Description Create a group, only accessible to users with staff permissions
// @Tags Groups
// @Accept json
// @Produce json
// @Param group body CreateGroupRequest true "Group to create"
// @Success 201 {object} models.Group
// @Failure 400 {object} response.ErrorResponse "Invalid request or scope not found"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Router /groups [post]
// @Security Bearer
func CreateGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Verify that the user is staff
    if !permissions.IsStaff(user) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionCreate)
        return
    }

    var req CreateGroupRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Verify that the scope exists
    var scopeExists bool
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Model(&models.Scope{}).
            Select("COUNT(*) > 0").
            Where("id = ?", req.ScopeID).
            Find(&scopeExists).Error
    })
    
    if err != nil || !scopeExists {
        response.Error(c, http.StatusBadRequest, ErrInvalidScopeIDs)
        return
    }

    // Create the group
    group := models.Group{
        Name:        req.Name,
        Description: req.Description,
        ScopeID:     req.ScopeID,
    }
    
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).Create(&group).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to create group")
        return
    }
    
    c.JSON(http.StatusCreated, group)
}

// DeleteGroup deletes a group and clears its associations with users and competitions
// @Summary Delete a group
// @Description Delete a group and clear all user and competition associations
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse "Group not found"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Failure 500 {object} response.ErrorResponse "Failed to delete group"
// @Router /groups/{group_id} [delete]
// @Security Bearer
func DeleteGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")
    
    // Only fetch the basic group data first for permission checking
    var group models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).Where("id = ?", groupID).First(&group).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
        return
    }

    if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

    // Use transaction handler to manage the transaction
    transactionHandler(c, func(tx *gorm.DB) error {
        // Check and clear user associations if needed
		userCount := tx.Model(&group).Association("Users").Count()
        
        if userCount > 0 {
            if err := tx.Model(&group).Association("Users").Clear(); err != nil {
                return err
            }
        }
        
        // Check and clear competition associations if needed
		compCount := tx.Model(&group).Association("Competitions").Count()
        
        if compCount > 0 {
            if err := tx.Model(&group).Association("Competitions").Clear(); err != nil {
                return err
            }
        }
        
        // Delete the group
        if err := tx.Delete(&group).Error; err != nil {
            return err
        }
        
        return nil
    })
    
    // If we got here without errors, return success
    c.Status(http.StatusNoContent)
}

// UpdateGroup updates a group's name and description
// @Summary Update a group name and description
// @Description Update a group's name and/or description fields
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param group body UpdateGroupRequest true "Group information to update"
// @Success 204 "No Content"
// @Failure 400 {object} response.ErrorResponse "Group not found or invalid request"
// @Failure 401 {object} response.ErrorResponse "Unauthorized access"
// @Failure 500 {object} response.ErrorResponse "Failed to update group"
// @Router /groups/{group_id} [put]
// @Security Bearer
func UpdateGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")
    
    // Validate request body first
    var req UpdateGroupRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    
    // Only fetch minimal data needed for permission check
    var group models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Select("id").
            Where("id = ?", groupID).
            First(&group).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrGroupNotFound)
        return
    }

    if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
        return
    }

    // Build updates map with only fields that need to change
    updates := map[string]interface{}{}
    if req.Name != "" {
        updates["name"] = req.Name
    }
    if req.Description != "" {
        updates["description"] = req.Description
    }

    // Only update if there are changes to make
    if len(updates) > 0 {
        err = withTimeout(func(ctx context.Context) error {
            return database.DB.WithContext(ctx).Model(&group).Updates(updates).Error
        })
        
        if err != nil {
            response.Error(c, http.StatusInternalServerError, "Failed to update group")
            return
        }
    }
    
    c.Status(http.StatusNoContent)
}

// GetGroupsFromScope retrieves all groups from a given scope
// @Summary Get all the groups from a given scope
// @Description Get all groups belonging to a scope and include their users
// @Tags Groups
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Success 200 {array} models.Group
// @Failure 400 {object} response.ErrorResponse "Scope not found or error fetching groups"
// @Router /groups/scope/{scope_id} [get]
func GetGroupsFromScope(c *gin.Context) {
    _, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    scopeID := c.Param("scope_id")
    
    // Verify scope exists
    var scopeExists bool
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Model(&models.Scope{}).
            Select("COUNT(*) > 0").
            Where("id = ?", scopeID).
            Find(&scopeExists).Error
    })
    
    if err != nil || !scopeExists {
        response.Error(c, http.StatusBadRequest, ErrScopeNotFound)
        return
    }

    // Get groups for this scope
    var groups []models.Group
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Where("scope_id = ?", scopeID).
            Preload("Users").
            Find(&groups).Error
    })
    
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrFetchingGroups)
        return
    }

    c.JSON(http.StatusOK, groups)
}

// GetMyGroups gets all groups that the authenticated user has access to
// @Summary Get all the groups that authenticated user has access to
// @Description Get all groups the authenticated user has access to based on their roles
// @Tags Groups
// @Accept json
// @Produce json
// @Success 200 {array} models.Group
// @Failure 400 {object} response.ErrorResponse "Error fetching groups"
// @Router /groups/me [get]
// @Security Bearer
func GetMyGroups(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    var groups []models.Group
    
    // Use a context with timeout for the raw SQL query
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    
    err = database.DB.WithContext(ctx).Raw(`
        SELECT DISTINCT g.*
        FROM public.groups g
        JOIN public.scopes s ON g.scope_id = s.id
        JOIN public.role_scopes rs ON rs.scope_id = s.id
        JOIN public.user_roles ur ON ur.role_id = rs.role_id
        WHERE ur.user_id = ?`, user.ID).Scan(&groups).Error
        
    if err != nil {
        response.Error(c, http.StatusBadRequest, ErrFetchingGroups)
        return
    }
    
    c.JSON(http.StatusOK, groups)
}