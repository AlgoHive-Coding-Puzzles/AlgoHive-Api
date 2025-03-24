package catalogs

import (
	"api/database"
	"api/services"
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

    cacheKey := "catalog_puzzle_details:" + catalogID + ":" + themeID + ":" + puzzleID
    ctx := c.Request.Context()

    // Try to get puzzle details from cache
    var puzzleDetails PuzzleResponse
    if found, _ := database.GetFromCache(ctx, cacheKey, &puzzleDetails); found {
        c.JSON(http.StatusOK, puzzleDetails)
        return
    }

    // Cache miss - fetch from database and API
    address, err := services.GetAddressFromCatalogId(catalogID)
    if err != nil {
        respondWithError(c, http.StatusNotFound, ErrCatalogNotFound)
        return
    }

    apiURL := address + "/theme?name=" + themeID
    var themeDetails ThemeResponse
    if err := services.CatalogProxyGet(apiURL, &themeDetails); err != nil {
        respondWithError(c, http.StatusInternalServerError, err.Error())
        return
    }

    puzzleIndex, err := strconv.Atoi(puzzleID)
    if err != nil || puzzleIndex < 0 || puzzleIndex >= len(themeDetails.Puzzles) {
        respondWithError(c, http.StatusBadRequest, "Invalid puzzle ID")
        return
    }
    puzzleDetails = themeDetails.Puzzles[puzzleIndex]

    // Cache the puzzle details
    _ = database.SetToCache(ctx, cacheKey, puzzleDetails)

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
    var req GetPuzzleInputRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        respondWithError(c, http.StatusBadRequest, "Invalid request body")
        return
    }

    ctx := c.Request.Context()
    puzzleInput, err := services.GetPuzzleInput(req.CatalogID, req.ThemeName, req.PuzzleID, req.SeedID, ctx)
    if err != nil {
        respondWithError(c, http.StatusInternalServerError, err.Error())
        return
    }
    
    c.JSON(http.StatusOK, puzzleInput)
}