package scopes

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var scopes []models.Scope
	if err := database.DB.Find(&scopes).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedGetScopes)
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
// @Success 200 
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /scopes/{id} [get]
// @Security Bearer
func GetScope(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	scopeID := c.Param("scope_id")
	var scope models.Scope
	
	if err := database.DB.Where("id = ?", scopeID).Preload("Catalogs").Preload("Roles").Preload("Groups").First(&scope).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrScopeNotFound)
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
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionCreate)
		return
	}
	
	var createScopeReq CreateScopeRequest
	if err := c.ShouldBindJSON(&createScopeReq); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check that catalogs exist
	var catalogs []*models.Catalog
	if err := database.DB.Where("id IN (?)", createScopeReq.CatalogsIds).Find(&catalogs).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
		return
	}

	if len(catalogs) != len(createScopeReq.CatalogsIds) {
		respondWithError(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
		return
	}

	// Create the scope
	scope := models.Scope{
		Name:        createScopeReq.Name,
		Description: createScopeReq.Description,
	}

	// Transaction to ensure atomicity
	tx := database.DB.Begin()

	if err := tx.Create(&scope).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedCreateScope+err.Error())
		return
	}

	if len(catalogs) > 0 {
		if err := tx.Model(&scope).Association("Catalogs").Append(catalogs); err != nil {
			tx.Rollback()
			respondWithError(c, http.StatusInternalServerError, ErrFailedAssociateAPIEnv+err.Error())
			return
		}
	}

	tx.Commit()
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
// @Router /scopes/{id} [put]
// @Security Bearer
func UpdateScope(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
		return
	}

	scopeID := c.Param("scope_id")
	var scope models.Scope
	if err := database.DB.Where("id = ?", scopeID).First(&scope).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrScopeNotFound)
		return
	}

	var updateScopeReq CreateScopeRequest
	if err := c.ShouldBindJSON(&updateScopeReq); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check that catalogs exist
	var catalogs []*models.Catalog
	if err := database.DB.Where("id IN (?)", updateScopeReq.CatalogsIds).Find(&catalogs).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
		return
	}

	if len(catalogs) != len(updateScopeReq.CatalogsIds) {
		respondWithError(c, http.StatusBadRequest, ErrInvalidAPIEnvIDs)
		return
	}

	// Transaction to ensure atomicity
	tx := database.DB.Begin()

	// Update the scope
	scope.Name = updateScopeReq.Name
	scope.Description = updateScopeReq.Description
	
	if err := tx.Save(&scope).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedUpdateScope+err.Error())
		return
	}

	// Update the associated catalogs
	if err := tx.Model(&scope).Association("Catalogs").Replace(catalogs); err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedUpdateAssoc+err.Error())
		return
	}

	tx.Commit()
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
// @Router /scopes/{id} [delete]
// @Security Bearer
func DeleteScope(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

    scopeID := c.Param("scope_id")
    var scope models.Scope
    if err := database.DB.Where("id = ?", scopeID).First(&scope).Error; err != nil {
        respondWithError(c, http.StatusNotFound, ErrScopeNotFound)
        return
    }

    // Check if any groups use this scope
    var groupCount int64
    if err := database.DB.Model(&models.Group{}).Where("scope_id = ?", scope.ID).Count(&groupCount).Error; err != nil {
        respondWithError(c, http.StatusInternalServerError, "Failed to check for associated groups: "+err.Error())
        return
    }

    if groupCount > 0 {
        respondWithError(c, http.StatusConflict, "Cannot delete scope: it is still being used by groups")
        return
    }

    // Transaction to ensure atomicity
    tx := database.DB.Begin()

    // Delete associations before deleting the scope
    if err := tx.Model(&scope).Association("Catalogs").Clear(); err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedClearAssoc+err.Error())
        return
    }

    // Delete role associations
    if err := tx.Model(&scope).Association("Roles").Clear(); err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedClearAssoc+err.Error())
        return
    }

    // Delete the scope
    if err := tx.Delete(&scope).Error; err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedDeleteScope+err.Error())
        return
    }

    tx.Commit()
    c.Status(http.StatusNoContent)
}