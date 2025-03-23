package catalogs

import (
	"api/database"
	"api/models"
	"encoding/json"
	"log"
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

// GetThemeDetailsFromCatalog récupère les détails d'un thème spécifique d'un catalogue
// @Summary Get details of a specific theme from a single API from it's ID
// @Description Get details of a specific theme from a single API from it's ID
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param catalogID path string true "API ID"
// @Param themeID path string true "Theme ID"
// @Success 200 {object} ThemeResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /catalogs/{catalogID}/themes/{themeID} [get]
// @Security Bearer
func GetThemeDetailsFromCatalog(c *gin.Context) {
    catalogID := c.Param("catalogID")
    themeID := c.Param("themeID")

    // Define a cache key specific to this theme's details
    cacheKey := "catalog_theme_details:" + catalogID + ":" + themeID
    ctx := c.Request.Context()

    // Try to get theme details from Redis cache first
    cachedThemeDetails, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit - parse and return the cached theme details
        var themeDetails ThemeResponse
        if err := json.Unmarshal([]byte(cachedThemeDetails), &themeDetails); err == nil {
            c.JSON(http.StatusOK, themeDetails)
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

    // Contacter l'API à l'adresse catalog.Address/themes/themeID
    apiURL := catalog.Address + "/theme?name=" + themeID

    log.Printf("Fetching theme details from API: %s", apiURL)
    
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

    var themeDetails ThemeResponse
    if err := json.NewDecoder(resp.Body).Decode(&themeDetails); err != nil {
        respondWithError(c, http.StatusInternalServerError, ErrDecodeResponseFailed)
        return
    }

    // Cache the theme details in Redis with 10 minute expiration
    themeDetailsJSON, err := json.Marshal(themeDetails)
    if err == nil {
        err = database.REDIS.Set(ctx, cacheKey, themeDetailsJSON, 10*time.Minute).Err()
        if err != nil {
            // Continue even if caching fails
        } else {
            // Optionally log the successful caching
        }
    }

    c.JSON(http.StatusOK, themeDetails)
}