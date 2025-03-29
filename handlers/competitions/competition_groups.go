package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
    GroupOperationTimeout = 5 * time.Second
)

// AddGroupToCompetition adds a group to a competition
// @Summary Add a group to a competition
// @Description Allow users in a group to access a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param group_id path string true "Group ID"
// @Success 200 {object} models.Competition
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/groups/{group_id} [post]
// @Security Bearer
func AddGroupToCompetition(c *gin.Context) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), GroupOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionManageGroups)
        return
    }

    competitionID := c.Param("id")
    groupID := c.Param("group_id")

    if competitionID == "" || groupID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition or group ID")
        return
    }

    // Start a transaction for atomicity
    tx := database.DB.WithContext(ctx).Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    var competition models.Competition
    if err := tx.First(&competition, "id = ?", competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Competition not found (ID: %s): %v", competitionID, err)
        response.Error(c, http.StatusNotFound, ErrCompetitionNotFound)
        return
    }

    var group models.Group
    if err := tx.First(&group, "id = ?", groupID).Error; err != nil {
        tx.Rollback()
        log.Printf("Group not found (ID: %s): %v", groupID, err)
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

    // Add the group to the competition with ON CONFLICT DO NOTHING for idempotent operations
    if err := tx.Exec("INSERT INTO competition_groups (group_id, competition_id) VALUES (?, ?) ON CONFLICT DO NOTHING", 
        groupID, competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Failed to add group %s to competition %s: %v", groupID, competitionID, err)
        response.Error(c, http.StatusInternalServerError, ErrFailedAddGroup)
        return
    }

    // Reload the competition with its associations
    if err := tx.Preload("Catalog").Preload("Groups").First(&competition, "id = ?", competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Failed to reload competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to reload competition")
        return
    }

    if err := tx.Commit().Error; err != nil {
        log.Printf("Failed to commit transaction: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to complete operation")
        return
    }

    // Invalidate any cached permissions for this competition
    cacheKeyPrefix := "perm_check:" + competitionID + ":*"
    if keys, err := database.REDIS.Keys(ctx, cacheKeyPrefix).Result(); err == nil && len(keys) > 0 {
        if err := database.REDIS.Del(ctx, keys...).Err(); err != nil {
            log.Printf("Failed to invalidate permission cache: %v", err)
        }
    }

    log.Printf("Group %s added to competition %s by user %s", groupID, competitionID, user.ID)
    c.JSON(http.StatusOK, competition)
}

// RemoveGroupFromCompetition removes a group from a competition
// @Summary Remove a group from a competition
// @Description Remove group access to a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param group_id path string true "Group ID"
// @Success 200 {object} models.Competition
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/groups/{group_id} [delete]
// @Security Bearer
func RemoveGroupFromCompetition(c *gin.Context) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), GroupOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionManageGroups)
        return
    }

    competitionID := c.Param("id")
    groupID := c.Param("group_id")

    if competitionID == "" || groupID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition or group ID")
        return
    }

    // Start a transaction for atomicity
    tx := database.DB.WithContext(ctx).Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    var competition models.Competition
    if err := tx.First(&competition, "id = ?", competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Competition not found (ID: %s): %v", competitionID, err)
        response.Error(c, http.StatusNotFound, ErrCompetitionNotFound)
        return
    }

    var group models.Group
    if err := tx.First(&group, "id = ?", groupID).Error; err != nil {
        tx.Rollback()
        log.Printf("Group not found (ID: %s): %v", groupID, err)
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

    // Remove the group from the competition
    if err := tx.Exec("DELETE FROM competition_groups WHERE group_id = ? AND competition_id = ?", 
        groupID, competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Failed to remove group %s from competition %s: %v", groupID, competitionID, err)
        response.Error(c, http.StatusInternalServerError, ErrFailedRemoveGroup)
        return
    }

    // Reload the competition with its associations
    if err := tx.Preload("Catalog").Preload("Groups").First(&competition, "id = ?", competitionID).Error; err != nil {
        tx.Rollback()
        log.Printf("Failed to reload competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to reload competition")
        return
    }

    if err := tx.Commit().Error; err != nil {
        log.Printf("Failed to commit transaction: %v", err)
        response.Error(c, http.StatusInternalServerError, "Failed to complete operation")
        return
    }

    // Invalidate any cached permissions for this competition
    cacheKeyPrefix := "perm_check:" + competitionID + ":*"
    if keys, err := database.REDIS.Keys(ctx, cacheKeyPrefix).Result(); err == nil && len(keys) > 0 {
        if err := database.REDIS.Del(ctx, keys...).Err(); err != nil {
            log.Printf("Failed to invalidate permission cache: %v", err)
        }
    }

    log.Printf("Group %s removed from competition %s by user %s", groupID, competitionID, user.ID)
    c.JSON(http.StatusOK, competition)
}

// GetCompetitionGroups retrieves all groups with access to a competition
// @Summary Get all groups with access to a competition
// @Description Get all groups that have access to the specified competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {array} models.Group
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/groups [get]
// @Security Bearer
func GetCompetitionGroups(c *gin.Context) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), GroupOperationTimeout)
    defer cancel()
    
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionManageGroups)
        return
    }

    competitionID := c.Param("id")
    if competitionID == "" {
        response.Error(c, http.StatusBadRequest, "Invalid competition ID")
        return
    }

    // Check cache for competition groups
    cacheKey := "competition_groups:" + competitionID
    var groups []models.Group
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedData != "" {
        if err := utils.UnmarshalJSON([]byte(cachedData), &groups); err == nil {
            c.JSON(http.StatusOK, groups)
            return
        }
        // If unmarshalling fails, continue to database query
        log.Printf("Failed to unmarshal cached groups: %v", err)
    }

    var competition models.Competition
    if err := database.DB.WithContext(ctx).First(&competition, "id = ?", competitionID).Error; err != nil {
        log.Printf("Competition not found (ID: %s): %v", competitionID, err)
        response.Error(c, http.StatusNotFound, ErrCompetitionNotFound)
        return
    }

    if err := database.DB.WithContext(ctx).
        Joins("JOIN competition_groups cg ON cg.group_id = groups.id").
        Where("cg.competition_id = ?", competitionID).
        Find(&groups).Error; err != nil {
        log.Printf("Failed to fetch groups for competition %s: %v", competitionID, err)
        response.Error(c, http.StatusInternalServerError, "Failed to fetch groups")
        return
    }

    // Cache the groups for future requests
    groupsJSON, err := utils.MarshalJSON(groups)
    if err == nil {
        if err := database.REDIS.Set(ctx, cacheKey, string(groupsJSON), 5*time.Minute).Err(); err != nil {
            log.Printf("Failed to cache competition groups: %v", err)
        }
    }

    c.JSON(http.StatusOK, groups)
}