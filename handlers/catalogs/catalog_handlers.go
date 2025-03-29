package catalogs

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

// Constants for error messages and timeouts
const (
    ErrFetchCatalogsFailed = "Failed to fetch catalogs"
    DatabaseTimeout        = 5 * time.Second
    CatalogsCacheKey       = "user_catalogs:"
    CatalogsCacheDuration  = 5 * time.Minute
)

// GetAllCatalogs Get all Catalogs
// @Summary Get all Catalogs Catalog
// @Description Get all Catalogs, only accessible to users with the API_ENV permission
// @Tags Catalogs
// @Accept json
// @Produce json
// @Success 200 {array} models.Catalog
// @Failure 401 {object} map[string]string
// @Router /catalogs [get]
// @Security Bearer
func GetAllCatalogs(c *gin.Context) {
    // Add timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), DatabaseTimeout)
    defer cancel()

    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Try to get from cache first
    cacheKey := CatalogsCacheKey + user.ID
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedData != "" {
        // Use cached result if available
        var catalogs []models.Catalog
        if err := utils.UnmarshalJSON([]byte(cachedData), &catalogs); err == nil {
            c.JSON(http.StatusOK, catalogs)
            return
        }
        // If unmarshalling fails, continue to fetch from database
    }
    
    var catalogs []models.Catalog
    // Check if user has admin permissions to see all catalogs
    if permissions.RolesHavePermission(user.Roles, permissions.CATALOGS) || 
       permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        if err := database.DB.WithContext(ctx).Preload("Scopes").Find(&catalogs).Error; err != nil {
            log.Printf("Error fetching all catalogs: %v", err)
            response.Error(c, http.StatusInternalServerError, ErrFetchCatalogsFailed)
            return
        }
    } else {
        if err := database.DB.WithContext(ctx).Raw(`
            SELECT DISTINCT c.*
            FROM public.catalogs c
            JOIN public.scope_catalogs sae ON sae.catalog_id = c.id
            JOIN public.role_scopes rs ON rs.scope_id = sae.scope_id
            JOIN public.user_roles ur ON ur.role_id = rs.role_id
            WHERE ur.user_id = ?`, user.ID).Scan(&catalogs).Error; err != nil {
            log.Printf("Error fetching user catalogs: %v", err)
            response.Error(c, http.StatusInternalServerError, ErrFetchCatalogsFailed)
            return
        }
    }
    
    // Cache the results for future requests
    catalogsJSON, err := utils.MarshalJSON(catalogs)
    if err == nil {
        if err := database.REDIS.Set(ctx, cacheKey, string(catalogsJSON), CatalogsCacheDuration).Err(); err != nil {
            // Just log the error, don't fail the request
            log.Printf("Failed to cache catalogs: %v", err)
        }
    }
    
    c.JSON(http.StatusOK, catalogs)
}