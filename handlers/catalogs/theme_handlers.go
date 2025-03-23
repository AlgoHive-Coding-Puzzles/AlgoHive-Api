package catalogs

import (
	"api/database"
	"api/models"
	"encoding/json"
	"net/http"
	"strconv"
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

// GetPuzzleFromThemeCatalog fetch the indexed puzzle from a theme, from a catalog
// @Summary Get the indexed puzzle from a theme, from a catalog
// @Description Get the indexed puzzle from a theme, from a catalog
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param catalogID path string true "API ID"
// @Param themeID path string true "Theme ID"
// @Param puzzleID path string true "Puzzle ID"
// @Success 200 {object} PuzzleResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /catalogs/{catalogID}/themes/{themeID}/puzzles/{puzzleID} [get]
// @Security Bearer
func GetPuzzleFromThemeCatalog(c *gin.Context) {
    catalogID := c.Param("catalogID")
    themeID := c.Param("themeID")
    puzzleID := c.Param("puzzleID")

    // Define a cache key specific to this puzzle's details
    cacheKey := "catalog_puzzle_details:" + catalogID + ":" + themeID + ":" + puzzleID
    ctx := c.Request.Context()

    // Try to get puzzle details from Redis cache first
    cachedPuzzleDetails, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit - parse and return the cached puzzle details
        var puzzleDetails PuzzleResponse
        if err := json.Unmarshal([]byte(cachedPuzzleDetails), &puzzleDetails); err == nil {
            c.JSON(http.StatusOK, puzzleDetails)
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

    // Contacter l'API à l'adresse catalog.Address/themes/
    apiURL := catalog.Address + "/theme?name=" + themeID
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

    // Récupérer le puzzle dans le thème à l'index donné theme.Puzzles[puzzleID]
    puzzleIndex, err := strconv.Atoi(puzzleID)
    if err != nil || puzzleIndex < 0 || puzzleIndex >= len(themeDetails.Puzzles) {
        respondWithError(c, http.StatusBadRequest, "Invalid puzzle ID")
        return
    }
    var puzzleDetails = themeDetails.Puzzles[puzzleIndex]
 
    c.JSON(http.StatusOK, puzzleDetails)
}