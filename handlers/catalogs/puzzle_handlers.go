package catalogs

import (
	"api/database"
	"api/services"
	"api/utils/response"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
    PuzzleCachePrefix = "catalog_puzzle_details:"
    PuzzleCacheTimeout = 30 * time.Minute
    PuzzleInputCachePrefix = "puzzle_input:"
)

// GetPuzzleFromThemeCatalog fetch the indexed puzzle from a theme, from a catalog
// @Summary Get the indexed puzzle from a theme, from a catalog
// @Description Get the indexed puzzle from a theme, from a catalog
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param catalogID path string true "API ID"
// @Param themeID path string true "Theme ID"
// @Param puzzleID path string true "Puzzle Index"
// @Success 200 {object} PuzzleResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /catalogs/{catalogID}/themes/{themeID}/puzzles/{puzzleIndex} [get]
// @Security Bearer
func GetPuzzleFromThemeCatalog(c *gin.Context) {
    catalogID := c.Param("catalogID")
    themeID := c.Param("themeID")
    puzzleIndex := c.Param("puzzleIndex")

    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), DatabaseTimeout)
    defer cancel()

    cacheKey := PuzzleCachePrefix + catalogID + ":" + themeID + ":" + puzzleIndex

    // Try to get puzzle details from cache
    var puzzleDetails PuzzleResponse
    if found, _ := database.GetFromCache(ctx, cacheKey, &puzzleDetails); found {
        c.JSON(http.StatusOK, puzzleDetails)
        return
    }

    // Cache miss - fetch from database and API
    address, err := services.GetAddressFromCatalogId(catalogID)
    if err != nil {
        log.Printf("Catalog not found (ID: %s): %v", catalogID, err)
        response.Error(c, http.StatusNotFound, ErrCatalogNotFound)
        return
    }

    apiURL := address + "/theme?name=" + themeID
    
    var themeDetails ThemeResponse
    if err := services.CatalogProxyGet(apiURL, &themeDetails); err != nil {
        log.Printf("Error fetching theme details from API: %v", err)
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    puzzleIndexInt, err := strconv.Atoi(puzzleIndex)
    if err != nil {
        log.Printf("Invalid puzzle index format: %s", puzzleIndex)
        response.Error(c, http.StatusBadRequest, "Invalid puzzle Index format")
        return
    }
    
    if puzzleIndexInt < 0 || puzzleIndexInt >= len(themeDetails.Puzzles) {
        log.Printf("Puzzle index out of bounds: %d (max: %d)", puzzleIndexInt, len(themeDetails.Puzzles)-1)
        response.Error(c, http.StatusBadRequest, "Puzzle index out of bounds")
        return
    }
    
    puzzleDetails = themeDetails.Puzzles[puzzleIndexInt]

    // Cache the puzzle details
    if err := database.SetToCacheWithExpiration(ctx, cacheKey, puzzleDetails, PuzzleCacheTimeout); err != nil {
        log.Printf("Failed to cache puzzle details: %v", err)
    }

    c.JSON(http.StatusOK, puzzleDetails)
}

// GetPuzzleInputFromThemeCatalog fetch the input of the indexed puzzle from a theme, from a catalog
// @Summary Get the input of the indexed puzzle from a theme, from a catalog
// @Description Get the input of the indexed puzzle from a theme, from a catalog
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param request body GetPuzzleInputRequest true "Request body"
// @Success 200 {object} PuzzleResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /catalogs/puzzle-input [post]
// @Security Bearer
func GetPuzzleInputFromThemeCatalog(c *gin.Context) {
    // Create a timeout context
    ctx, cancel := context.WithTimeout(c.Request.Context(), DatabaseTimeout)
    defer cancel()

    var req GetPuzzleInputRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        log.Printf("Invalid puzzle input request: %v", err)
        response.Error(c, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Validate the request fields
    if req.CatalogID == "" || req.ThemeName == "" || req.PuzzleID == "" || req.SeedID == "" {
        log.Printf("Missing required fields in puzzle input request")
        response.Error(c, http.StatusBadRequest, "Missing required fields in request")
        return
    }

    // Create a cache key for this specific input request
    cacheKey := PuzzleInputCachePrefix + req.CatalogID + ":" + req.ThemeName + ":" + req.PuzzleID + ":" + req.SeedID
    
    // Check if we have a cached response
    var cachedInput interface{}
    if found, _ := database.GetFromCache(ctx, cacheKey, &cachedInput); found {
        c.JSON(http.StatusOK, cachedInput)
        return
    }
    
    // No cache hit, need to fetch the puzzle input
    puzzleInput, err := services.GetPuzzleInput(
        req.CatalogID, 
        req.ThemeName, 
        req.PuzzleID, 
        req.SeedID, 
        ctx,
    )
    if err != nil {
        log.Printf("Error fetching puzzle input: %v", err)
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    
    // Cache the result for future requests
    if err := database.SetToCacheWithExpiration(ctx, cacheKey, puzzleInput, PuzzleCacheTimeout); err != nil {
        log.Printf("Failed to cache puzzle input: %v", err)
    }
    
    c.JSON(http.StatusOK, puzzleInput)
}