package groups

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllGroups retrieves all groups
// @Summary Get all groups
// @Description Get all groups, only accessible to users with the GROUPS permission
// @Tags Groups
// @Accept json
// @Produce json
// @Success 200 {array} models.Group
// @Failure 401 {object} map[string]string
// @Router /groups [get]
// @Security Bearer
func GetAllGroups(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !permissions.RolesHavePermission(user.Roles, permissions.GROUPS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var groups []models.Group
	if err := database.DB.Find(&groups).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFetchingGroups)
		return
	}
	
	c.JSON(http.StatusOK, groups)
}

// GetGroup retrieves a group by ID
// @Summary Get a group
// @Description Get a group
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Success 200 {object} models.Group
// @Failure 400 {object} map[string]string
// @Router /groups/{group_id} [get]
// @Security Bearer
func GetGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).Preload("Users").Preload("Competitions").First(&group).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
		return
	}

	if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionViewGroup)
		return
	}

	c.JSON(http.StatusOK, group)
}

// CreateGroup creates a new group
// @Summary Create a group
// @Description Create a group, only accessible to users with the GROUPS permission
// @Tags Groups
// @Accept json
// @Produce json
// @Param group body CreateGroupRequest true "Group to create"
// @Success 201 {object} models.Group
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /groups [post]
// @Security Bearer
func CreateGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Verify that the user is staff
	if !permissions.IsStaff(user) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionCreate)
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Verify that the scope exists
	var scopes []models.Scope
	if err := database.DB.Where("id = ?", req.ScopeId).Find(&scopes).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidScopeIDs)
		return
	}
	
	if len(scopes) == 0 {
		respondWithError(c, http.StatusBadRequest, ErrInvalidScopeIDs)
		return
	}

	// Create the group
	group := models.Group{
		Name:        req.Name,
		Description: req.Description,
		ScopeID:     req.ScopeId,
	}
	
	if err := database.DB.Create(&group).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to create group")
		return
	}
	
	c.JSON(http.StatusCreated, group)
}

// DeleteGroup deletes a group and cascades deletion of all its users and competitions
// @Summary Delete a group and cascade delete all users and competitions associated with the group
// @Description Delete a group and cascade delete all users and competitions associated with the group
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Success 204 {object} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /groups/{group_id} [delete]
// @Security Bearer
func DeleteGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).Preload("Users").Preload("Competitions").First(&group).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
		return
	}

	if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDelete)
		return
	}

	// Delete the group
	tx := database.DB.Begin()
	
	// First remove the associations
	if err := tx.Model(&group).Association("Users").Clear(); err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, "Failed to clear group associations")
		return
	}
	
	if err := tx.Model(&group).Association("Competitions").Clear(); err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, "Failed to clear group competitions")
		return
	}
	
	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, "Failed to delete group")
		return
	}
	
	tx.Commit()
	c.Status(http.StatusNoContent)
}

// UpdateGroup updates a group's name and description
// @Summary Update a group name and description
// @Description Update a group name and description
// @Tags Groups
// @Accept json
// @Produce json
// @Param group_id path string true "Group ID"
// @Param group body UpdateGroupRequest true "Group to update"
// @Success 204 {object} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /groups/{group_id} [put]
// @Security Bearer
func UpdateGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
		return
	}

	if !userCanManageGroup(user.ID, &group) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Update only non-empty fields
	updates := models.Group{}
	if req.Name != "" {
		updates.Name = req.Name
	}
	if req.Description != "" {
		updates.Description = req.Description
	}

	if err := database.DB.Model(&group).Updates(updates).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update group")
		return
	}
	
	c.Status(http.StatusNoContent)
}

// GetGroupsFromScope retrieves all groups from a given scope
// @Summary Get all the groups from a given scope
// @Description Get all the groups from a given scope and list the users in each group
// @Tags Groups
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Success 200 {array} models.Group
// @Failure 400 {object} map[string]string
// @Router /groups/scope/{scope_id} [get]
func GetGroupsFromScope(c *gin.Context) {
	_, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	scopeID := c.Param("scope_id")
	var scope models.Scope
	if err := database.DB.Where("id = ?", scopeID).First(&scope).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrScopeNotFound)
		return
	}

	var groups []models.Group
	if err := database.DB.Where("scope_id = ?", scopeID).Preload("Users").Find(&groups).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrFetchingGroups)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// Get all the groups that authenticated user has access to
// @Summary Get all the groups that authenticated user has access to
// @Description Get all the groups that authenticated user has access to
// @Tags Groups
// @Accept json
// @Produce json
// @Success 200 {array} models.Group
// @Failure 400 {object} map[string]string
// @Router /groups/me [get]
// @Security Bearer
func GetMyGroups(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	var groups []models.Group
	if err := database.DB.Raw(`
		SELECT DISTINCT g.*
		FROM public.groups g
		JOIN public.scopes s ON g.scope_id = s.id
		JOIN public.role_scopes rs ON rs.scope_id = s.id
		JOIN public.user_roles ur ON ur.role_id = rs.role_id
		WHERE ur.user_id = ?`, user.ID).Scan(&groups).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrFetchingGroups)
		return
	}
	c.JSON(http.StatusOK, groups)
}