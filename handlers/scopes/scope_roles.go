package scopes

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
    // roleOperationTimeout defines the standard timeout for role operations
    roleOperationTimeout = 5 * time.Second
)

// AttachScopeToRole attaches a scope to a role
// @Summary Attach the scope to a role
// @Description Attach the scope to a role, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /scopes/{scope_id}/roles/{role_id} [post]
// @Security Bearer
func AttachScopeToRole(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionAttach)
        return
    }

    scopeID := c.Param("scope_id")
    roleID := c.Param("role_id")

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), roleOperationTimeout)
    defer cancel()

    // Begin transaction for consistency
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

    // Find scope
    var scope models.Scope
    if err := tx.Where("id = ?", scopeID).First(&scope).Error; err != nil {
        tx.Rollback()
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrScopeNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error finding scope: "+err.Error())
        }
        return
    }

    // Find role
    var role models.Role
    if err := tx.Where("id = ?", roleID).First(&role).Error; err != nil {
        tx.Rollback()
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error finding role: "+err.Error())
        }
        return
    }

    // Check if association already exists to avoid duplicates
    var count int64
    if err := tx.Raw("SELECT COUNT(*) FROM role_scopes WHERE role_id = ? AND scope_id = ?", 
        role.ID, scope.ID).Count(&count).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Error checking existing association: "+err.Error())
        return
    }

    if count == 0 {
        // Add association
        if err := tx.Model(&scope).Association("Roles").Append(&role); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedAttachRole+err.Error())
            return
        }
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    // Reload scope with all relationships for response
    if err := database.DB.
        Preload("Roles").
        Preload("Catalogs").
        Preload("Groups").
        First(&scope, scope.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Association created but failed to retrieve details")
        return
    }

    c.JSON(http.StatusOK, scope)
}

// DetachScopeFromRole detaches a scope from a role
// @Summary Detach the scope from a role
// @Description Detach the scope from a role, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param scope_id path string true "Scope ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /scopes/{scope_id}/roles/{role_id} [delete]
// @Security Bearer
func DetachScopeFromRole(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionDetach)
        return
    }

    scopeID := c.Param("scope_id")
    roleID := c.Param("role_id")

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), roleOperationTimeout)
    defer cancel()

    // Begin transaction
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

    // Find scope
    var scope models.Scope
    if err := tx.Where("id = ?", scopeID).First(&scope).Error; err != nil {
        tx.Rollback()
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrScopeNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error finding scope: "+err.Error())
        }
        return
    }

    // Find role
    var role models.Role
    if err := tx.Where("id = ?", roleID).First(&role).Error; err != nil {
        tx.Rollback()
        if err == gorm.ErrRecordNotFound {
            response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        } else {
            response.Error(c, http.StatusInternalServerError, "Error finding role: "+err.Error())
        }
        return
    }

    // Remove the association
    if err := tx.Model(&scope).Association("Roles").Delete(&role); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedDetachRole+err.Error())
        return
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
        return
    }

    // Reload scope with all relationships for response
    if err := database.DB.
        Preload("Roles").
        Preload("Catalogs").
        Preload("Groups").
        First(&scope, scope.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Association removed but failed to retrieve details")
        return
    }

    c.JSON(http.StatusOK, scope)
}

// GetRoleScopes retrieves all scopes that a role has access to
// @Summary Get all the scopes that a role has access to
// @Description Get all the scopes that a role has access to, only accessible to users with the SCOPES permission
// @Tags Scopes
// @Accept json
// @Produce json
// @Param roles query []string true "Roles IDs"
// @Success 200 {array} models.Scope
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /scopes/roles [get]
// @Security Bearer
func GetRoleScopes(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    if !permissions.IsStaff(user) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    // Get role IDs from the request parameters
    rolesParam := c.QueryArray("roles")
    
    // Process role IDs - handle comma-separated values
    var roles []string
    if len(rolesParam) == 1 && strings.Contains(rolesParam[0], ",") {
        roles = strings.Split(rolesParam[0], ",")
        // Trim whitespace from IDs
        for i, role := range roles {
            roles[i] = strings.TrimSpace(role)
        }
    } else {
        roles = rolesParam
    }

    if len(roles) == 0 {
        response.Error(c, http.StatusBadRequest, ErrRolesRequired)
        return
    }

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), roleOperationTimeout)
    defer cancel()
    db := database.DB.WithContext(ctx)

    // Load roles to check permissions
    var loadedRoles []*models.Role
    if err := db.Where("id IN ?", roles).Find(&loadedRoles).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to get roles: "+err.Error())
        return
    }

    // If no roles found, return empty result
    if len(loadedRoles) == 0 {
        c.JSON(http.StatusOK, []models.Scope{})
        return
    }

    // If roles have the OWNER permission, return all scopes
    if permissions.RolesHavePermission(loadedRoles, permissions.OWNER) {
        var scopes []models.Scope
        if err := db.Preload("Groups").Preload("Catalogs").Find(&scopes).Error; err != nil {
            response.Error(c, http.StatusInternalServerError, ErrFailedGetScopes+": "+err.Error())
            return
        }

        c.JSON(http.StatusOK, scopes)
        return
    }

    // Otherwise, retrieve only the scopes associated with the specified roles
    var scopeIDs []string
    if err := db.Raw(`
        SELECT DISTINCT s.id
        FROM scopes s
        JOIN role_scopes rs ON s.id = rs.scope_id
        JOIN roles r ON rs.role_id = r.id
        WHERE r.id IN ?
    `, roles).Pluck("id", &scopeIDs).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedGetScopes+": "+err.Error())
        return
    }

    // Return empty array if no scopes found
    if len(scopeIDs) == 0 {
        c.JSON(http.StatusOK, []models.Scope{})
        return
    }

    // Load scopes with their relationships
    var scopes []models.Scope
    if err := db.Where("id IN ?", scopeIDs).
        Preload("Groups").
        Preload("Catalogs").
        Find(&scopes).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedGetScopes+": "+err.Error())
        return
    }

    c.JSON(http.StatusOK, scopes)
}