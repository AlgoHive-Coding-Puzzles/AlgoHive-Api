package catalogs

import (
	"api/database"
	"api/models"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

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

// func GetPuzzleInputFromThemeCatalog(c *gin.Context) {

// GetPuzzleInputFromThemeCatalog fetch the input of the indexed puzzle from a theme, from a catalog
// @Summary Get the input of the indexed puzzle from a theme, from a catalog
// @Description Get the input of the indexed puzzle from a theme, from a catalog
// @Tags Catalogs
// @Accept json
// @Produce json
// @Param catalogID path string true "API ID"
// @Param themeID path string true "Theme ID"
// @Param puzzleID path string true "Puzzle ID"
// @Param inputID path string true "Input ID"
// @Success 200 {object} PuzzleResponse
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /catalogs/{catalogID}/themes/{themeID}/puzzles/{puzzleID}/inputs/{inputID} [get]
// @Security Bearer
func GetPuzzleInputFromThemeCatalog(c *gin.Context) {
    catalogID := c.Param("catalogID")
    themeID := c.Param("themeID")
    puzzleID := c.Param("puzzleID")
    inputID := c.Param("inputID")

    log.Println("Catalog ID:", catalogID)
    log.Println("Theme ID:", themeID)
    log.Println("Puzzle ID:", puzzleID)
    log.Println("Input ID:", inputID)

    // Define a cache key specific to this puzzle input's details
    cacheKey := "catalog_puzzle_input:" + catalogID + ":" + themeID + ":" + puzzleID + ":" + inputID
    ctx := c.Request.Context()

    // Try to get puzzle input details from Redis cache first
    cachedInputDetails, err := database.REDIS.Get(ctx, cacheKey).Result()
    if err == nil {
        // Cache hit - parse and return the cached input details
        var inputDetails map[string]interface{}
        if err := json.Unmarshal([]byte(cachedInputDetails), &inputDetails); err == nil {
            c.JSON(http.StatusOK, inputDetails)
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

    log.Println("Catalog address:", catalog.Address)

    // Construct the API URL to fetch puzzle input
    apiURL := catalog.Address + "/puzzle/generate/input?theme=" + themeID + "&puzzle=" + puzzleID + "&unique_id=" + inputID
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

    // Decode the response body
    var inputDetails map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&inputDetails); err != nil {
        respondWithError(c, http.StatusInternalServerError, ErrDecodeResponseFailed)
        return
    }

    // Cache the input details for future requests
    inputDetailsJSON, err := json.Marshal(inputDetails)
    if err == nil {
        _ = database.REDIS.Set(ctx, cacheKey, inputDetailsJSON, 0).Err()
    }

    // Return the input details as JSON
    c.JSON(http.StatusOK, inputDetails)
}