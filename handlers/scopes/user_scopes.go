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
)

const (
    // userScopesQueryTimeout defines the standard timeout for user scopes queries
    userScopesQueryTimeout = 5 * time.Second
)

// GetUserScopes retrieves all scopes that the user has access to
// @Summary Get all scopes that the user has access to
// @Description Get all scopes that the user has access to (based on roles) if the user has the SCOPES permission we return all scopes
// @Tags Scopes
// @Accept json
// @Produce json
// @Success 200 {array} models.Scope
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /scopes/user [get]
// @Security Bearer
func GetUserScopes(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), userScopesQueryTimeout)
    defer cancel()

    var scopes []models.Scope
    
    // If the user has the SCOPES permission, return all scopes
    if permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
        if err := database.DB.WithContext(ctx).
            Preload("Catalogs").
            Preload("Roles").
            Preload("Groups").
            Find(&scopes).Error; err != nil {
            response.Error(c, http.StatusInternalServerError, ErrFailedGetScopes)
            return
        }
    } else {
        // Otherwise, retrieve only the scopes they have access to via their roles
        // We'll use a more efficient query to avoid loading unnecessary data
        var scopeIDs []string
        
        err := database.DB.WithContext(ctx).Raw(`
            SELECT DISTINCT s.id
            FROM scopes s
            JOIN role_scopes rs ON s.id = rs.scope_id
            JOIN roles r ON rs.role_id = r.id
            JOIN user_roles ur ON r.id = ur.role_id
            WHERE ur.user_id = ?
        `, user.ID).Pluck("id", &scopeIDs).Error
        
        if err != nil {
            response.Error(c, http.StatusInternalServerError, "Failed to load user scopes: "+err.Error())
            return
        }
        
        if len(scopeIDs) > 0 {
            if err := database.DB.WithContext(ctx).
                Preload("Catalogs").
                Preload("Groups").
                Where("id IN ?", scopeIDs).
                Find(&scopes).Error; err != nil {
                response.Error(c, http.StatusInternalServerError, "Failed to load scopes data: "+err.Error())
                return
            }
        }
    }

    c.JSON(http.StatusOK, scopes)
}