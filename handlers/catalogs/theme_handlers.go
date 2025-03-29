package catalogs

import (
	"api/config"
	"api/database"
	"api/models"
	"api/utils/response"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
    ThemeCachePrefix    = "catalog_themes:"
    ThemeDetailsCachePrefix = "catalog_theme_details:"
    ThemeCacheTimeout   = 10 * time.Minute
)

// GetThemesFromCatalog Fetch all the themes from a single API from its ID
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

    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), DatabaseTimeout)
    defer cancel()

    // Define a cache key specific to this catalog's themes
    cacheKey := ThemeCachePrefix + catalogID

    // Try to get themes from Redis cache first
    cachedThemes, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedThemes != "" {
        // Cache hit - parse and return the cached themes
        var themes []ThemeResponse
        if err := json.Unmarshal([]byte(cachedThemes), &themes); err == nil {
            c.JSON(http.StatusOK, themes)
            return
        }
        // If unmarshaling fails, continue with fetching from API
        log.Printf("Failed to unmarshal cached themes: %v", err)
    }

    // Cache miss or error - fetch from database and API
    var catalog models.Catalog
    if err := database.DB.WithContext(ctx).First(&catalog, "id = ?", catalogID).Error; err != nil {
        log.Printf("Catalog not found (ID: %s): %v", catalogID, err)
        response.Error(c, http.StatusNotFound, ErrCatalogNotFound)
        return
    }

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: 5 * time.Second,
    }

    // Contact the API at the address catalog.Address/themes
    apiURL := catalog.Address + "/themes"
    
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
    if err != nil {
        log.Printf("Error creating request for %s: %v", apiURL, err)
        response.Error(c, http.StatusInternalServerError, ErrAPIReachFailed)
        return
    }
    
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error reaching API %s: %v", apiURL, err)
        response.Error(c, http.StatusInternalServerError, ErrAPIReachFailed)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("API returned non-200 status: %d", resp.StatusCode)
        response.Error(c, resp.StatusCode, ErrAPIReachFailed)
        return
    }

    var themes []ThemeResponse
    if err := json.NewDecoder(resp.Body).Decode(&themes); err != nil {
        log.Printf("Error decoding API response: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrDecodeResponseFailed)
        return
    }

    // Cache the themes in Redis with appropriate expiration
    themesJSON, err := json.Marshal(themes)
    if err == nil {
        cacheExpiration := config.RedisExpiMin * time.Minute
        if cacheExpiration == 0 {
            cacheExpiration = ThemeCacheTimeout
        }
        
        err = database.REDIS.Set(ctx, cacheKey, themesJSON, cacheExpiration).Err()
        if err != nil {
            // Just log the error and continue
            log.Printf("Failed to cache themes: %v", err)
        }
    }

    c.JSON(http.StatusOK, themes)
}

// GetThemeDetailsFromCatalog fetch details of a specific theme from a single API from its ID
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

    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), DatabaseTimeout)
    defer cancel()

    // Define a cache key specific to this theme's details
    cacheKey := ThemeDetailsCachePrefix + catalogID + ":" + themeID

    // Try to get theme details from Redis cache first
    cachedThemeDetails, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil && cachedThemeDetails != "" {
        // Cache hit - parse and return the cached theme details
        var themeDetails ThemeResponse
        if err := json.Unmarshal([]byte(cachedThemeDetails), &themeDetails); err == nil {
            c.JSON(http.StatusOK, themeDetails)
            return
        }
        // If unmarshaling fails, continue with fetching from API
        log.Printf("Failed to unmarshal cached theme details: %v", err)
    }

    // Cache miss or error - fetch from database and API
    var catalog models.Catalog
    if err := database.DB.WithContext(ctx).First(&catalog, "id = ?", catalogID).Error; err != nil {
        log.Printf("Catalog not found (ID: %s): %v", catalogID, err)
        response.Error(c, http.StatusNotFound, ErrCatalogNotFound)
        return
    }

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: 5 * time.Second,
    }

    // Contact the API at the address catalog.Address/theme?name=themeID
    apiURL := catalog.Address + "/theme?name=" + themeID
    
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
    if err != nil {
        log.Printf("Error creating request for %s: %v", apiURL, err)
        response.Error(c, http.StatusInternalServerError, ErrAPIReachFailed)
        return
    }
    
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error reaching API %s: %v", apiURL, err)
        response.Error(c, http.StatusInternalServerError, ErrAPIReachFailed)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("API returned non-200 status: %d", resp.StatusCode)
        response.Error(c, resp.StatusCode, ErrAPIReachFailed)
        return
    }

    var themeDetails ThemeResponse
    if err := json.NewDecoder(resp.Body).Decode(&themeDetails); err != nil {
        log.Printf("Error decoding API response: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrDecodeResponseFailed)
        return
    }

    // Cache the theme details in Redis with appropriate expiration
    themeDetailsJSON, err := json.Marshal(themeDetails)
    if err == nil {
        cacheExpiration := config.RedisExpiMin * time.Minute
        if cacheExpiration == 0 {
            cacheExpiration = ThemeCacheTimeout
        }
        
        err = database.REDIS.Set(ctx, cacheKey, themeDetailsJSON, cacheExpiration).Err()
        if err != nil {
            // Just log the error and continue
            log.Printf("Failed to cache theme details: %v", err)
        }
    }

    c.JSON(http.StatusOK, themeDetails)
}