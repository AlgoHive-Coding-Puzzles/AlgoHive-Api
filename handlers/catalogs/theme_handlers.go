package catalogs

import (
	"api/database"
	"api/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetThemesFromCatalog récupère tous les thèmes d'un catalogue
// @Summary Get all the themes from a single API from it's ID
// @Description Get all the themes from a single API from it's ID
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param catalogID path string true "API ID"
// @Success 200 {array} ThemeResponse
// @Failure 401 {object} map[string]string
// @Router /catalogs/{catalogID}/themes [get]
// @Security Bearer
func GetThemesFromCatalog(c *gin.Context) {
    catalogID := c.Param("catalogID")

    // Define a cache key specific to this catalog's themes
    cacheKey := "catalog_themes:" + catalogID
    ctx := c.Request.Context()

    // Try to get themes from Redis cache first
    cachedThemes, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit - parse and return the cached themes
        var themes []ThemeResponse
        if err := json.Unmarshal([]byte(cachedThemes), &themes); err == nil {
            c.JSON(http.StatusOK, themes)
            return
        }
        // If unmarshaling fails, continue with fetching from API
    }

    // Cache miss or error - fetch from database and API
    var catalog models.Catalog
    if err := database.DB.First(&catalog, "id = ?", catalogID).Error; err != nil {
        respondWithError(c, http.StatusNotFound, ErrCatalogNotFound)
        return
    }

    // Contacter l'API à l'adresse catalog.Address/themes
    apiURL := catalog.Address + "/themes"
    
    resp, err := http.Get(apiURL)
    if err != nil {
        respondWithError(c, http.StatusInternalServerError, ErrAPIReachFailed)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        respondWithError(c, resp.StatusCode, ErrAPIReachFailed)
        return
    }

    var themes []ThemeResponse
    if err := json.NewDecoder(resp.Body).Decode(&themes); err != nil {
        respondWithError(c, http.StatusInternalServerError, ErrDecodeResponseFailed)
        return
    }

    // Cache the themes in Redis with 10 minute expiration
    themesJSON, err := json.Marshal(themes)
    if err == nil {
        err = database.REDIS.Set(ctx, cacheKey, themesJSON, 10*time.Minute).Err()
        if err != nil {
            // Continue even if caching fails
        } else {
        }
    }

    c.JSON(http.StatusOK, themes)
}