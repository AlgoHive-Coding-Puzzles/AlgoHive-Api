package scopes

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

const (
    // defaultQueryTimeout defines the standard timeout for database operations
    defaultQueryTimeout = 5 * time.Second
)

// withTimeout executes a database function with a timeout context
func withTimeout(operation func(ctx context.Context) error) error {
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    return operation(ctx)
}

// GetAllScopes retrieves all scopes
// @Summary Get all scopes
// @Description Get all scopes, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Success 200 {array} models.Scope
// @Failure 401 {object} map[string]string
// @Router /scopes [get]
// @Security Bearer
func GetAllScopes(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    var scopes []models.Scope
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Preload("Catalogs").
            Preload("Roles").
            Preload("Groups").
            Find(&scopes).Error
    })

    if err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedGetScopes)
        return
    }

    c.JSON(http.StatusOK, scopes)
}

// GetScope retrieves a scope by ID
// @Summary Get a scope
// @Description Get a scope, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Success 200 {object} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /scopes/{scope_id} [get]
// @Security Bearer
func GetScope(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    scopeID := c.Param("scope_id")
    var scope models.Scope
    
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).
            Where("id = ?", scopeID).
            Preload("Catalogs").
            Preload("Roles").
            Preload("Groups").
            First(&scope).Error
    })
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrScopeNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error fetching scope: "+err.Error())
        }
        return
    }

    c.JSON(http.StatusOK, scope)
}

// CreateScope creates a new scope
// @Summary Create a scope
// @Description Create a scope, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param createScope body CreateScopeRequest true "Scope Details"
// @Success 201 {object} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /scopes [post]
// @Security Bearer
func CreateScope(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionCreate)
        return
    }
    
    var createScopeReq CreateScopeRequest
    if err := c.ShouldBindJSON(&createScopeReq); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Start transaction early to ensure consistent timeout handling
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    tx := database.DB.WithContext(ctx).Begin()
    
    if tx.Error != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to begin transaction: "+tx.Error.Error())
        return
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Check that catalogs exist
    var catalogCount int64
    if err := tx.Model(&models.Catalog{}).
        Where("id IN (?)", createScopeReq.CatalogsIds).
        Count(&catalogCount).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to check catalogs: "+err.Error())
        return
    }

    if int(catalogCount) != len(createScopeReq.CatalogsIds) {
        tx.Rollback()
        response.Error(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
        return
    }

    // Create the scope
    scope := models.Scope{
        Name:        createScopeReq.Name,
        Description: createScopeReq.Description,
    }

    if err := tx.Create(&scope).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedCreateScope+err.Error())
        return
    }

    // Only load catalogs if we need them
    if len(createScopeReq.CatalogsIds) > 0 {
        var catalogs []*models.Catalog
        if err := tx.Where("id IN (?)", createScopeReq.CatalogsIds).Find(&catalogs).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrInvalidAPIEnvIDs+": "+err.Error())
            return
        }

        if err := tx.Model(&scope).Association("Catalogs").Append(catalogs); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAssociateAPIEnv+err.Error())
            return
        }
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    // Reload the scope with associations for response
    if err := database.DB.Preload("Catalogs").First(&scope, scope.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Scope created but failed to retrieve details: "+err.Error())
        return
    }

    c.JSON(http.StatusCreated, scope)
}

// UpdateScope updates an existing scope
// @Summary Update a scope
// @Description Update a scope, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Param updateScope body CreateScopeRequest true "Scope Details"
// @Success 200 {object} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /scopes/{scope_id} [put]
// @Security Bearer
func UpdateScope(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }
    
    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
        return
    }

    scopeID := c.Param("scope_id")
    var scope models.Scope
    
    // First check if the scope exists
    err = withTimeout(func(ctx context.Context) error {
        return database.DB.WithContext(ctx).Where("id = ?", scopeID).First(&scope).Error
    })
    
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrScopeNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error finding scope: "+err.Error())
        }
        return
    }

    var updateScopeReq CreateScopeRequest
    if err := c.ShouldBindJSON(&updateScopeReq); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Start transaction for update operations
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    tx := database.DB.WithContext(ctx).Begin()
    
    if tx.Error != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to begin transaction: "+tx.Error.Error())
        return
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Check that all catalogs exist
    var catalogCount int64
    if err := tx.Model(&models.Catalog{}).
        Where("id IN (?)", updateScopeReq.CatalogsIds).
        Count(&catalogCount).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to check catalogs: "+err.Error())
        return
    }

    if int(catalogCount) != len(updateScopeReq.CatalogsIds) {
        tx.Rollback()
        response.Error(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
        return
    }

    // Update the scope
    scope.Name = updateScopeReq.Name
    scope.Description = updateScopeReq.Description
    
    if err := tx.Save(&scope).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedUpdateScope+err.Error())
        return
    }

    // Load and update catalog associations
    var catalogs []*models.Catalog
    if err := tx.Where("id IN (?)", updateScopeReq.CatalogsIds).Find(&catalogs).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrInvalidAPIEnvIDs+": "+err.Error())
        return
    }
    
    // Replace catalog associations
    if err := tx.Model(&scope).Association("Catalogs").Replace(catalogs); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedUpdateAssoc+err.Error())
        return
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    // Reload scope with all associations for response
    if err := database.DB.Preload("Catalogs").Preload("Roles").Preload("Groups").First(&scope, scope.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Scope updated but failed to retrieve details")
        return
    }
    
    c.JSON(http.StatusOK, scope)
}

// DeleteScope deletes a scope
// @Summary Delete a scope
// @Description Delete a scope, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Success 204 {object} string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /scopes/{scope_id} [delete]
// @Security Bearer
func DeleteScope(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

    scopeID := c.Param("scope_id")
    
    // Start transaction with timeout
    ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
    defer cancel()
    tx := database.DB.WithContext(ctx).Begin()
    
    if tx.Error != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to begin transaction: "+tx.Error.Error())
        return
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // First load the scope to check existence
    var scope models.Scope
    if err := tx.Where("id = ?", scopeID).First(&scope).Error; err != nil {
        tx.Rollback()
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrScopeNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Failed to find scope: "+err.Error())
        }
        return
    }

    // Check if any groups use this scope
    var groupCount int64
    if err := tx.Model(&models.Group{}).Where("scope_id = ?", scope.ID).Count(&groupCount).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to check for associated groups: "+err.Error())
        return
    }

    if groupCount > 0 {
        tx.Rollback()
        response.Error(c, http.StatusConflict, "Cannot delete scope: it is still being used by groups")
        return
    }

    // Delete associations before deleting the scope
    if err := tx.Model(&scope).Association("Catalogs").Clear(); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedClearAssoc+err.Error())
        return
    }

    // Delete role associations
    if err := tx.Model(&scope).Association("Roles").Clear(); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedClearAssoc+err.Error())
        return
    }

    // Delete the scope
    if err := tx.Delete(&scope).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedDeleteScope+err.Error())
        return
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    c.Status(http.StatusNoContent)
}