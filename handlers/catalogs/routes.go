package catalogs

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to catalogs
// r: the RouterGroup to which the routes are added
func RegisterRoutes(r *gin.RouterGroup) {
    // Create rate limiters for potentially expensive endpoints
    catalogRateLimiter := middleware.NewRateLimiter(15000, 1000) // 20 requests per minute with burst capacity
	puzzleInputRateLimiter := middleware.NewRateLimiter(15000, 1000) // 10 requests per minute with burst capacity
    
    // Create catalogs group and apply authentication middleware
    catalogs := r.Group("/catalogs")
    catalogs.Use(middleware.AuthMiddleware())
    {
        // Apply rate limiting to API endpoints
        catalogs.GET("/", middleware.RateLimiterMiddleware(catalogRateLimiter), GetAllCatalogs)
        catalogs.GET("/:catalogID/themes", GetThemesFromCatalog)
        catalogs.GET("/:catalogID/themes/:themeID", GetThemeDetailsFromCatalog)
        catalogs.GET("/:catalogID/themes/:themeID/puzzles/:puzzleIndex", GetPuzzleFromThemeCatalog)
        catalogs.POST("/puzzle-input", middleware.RateLimiterMiddleware(puzzleInputRateLimiter), GetPuzzleInputFromThemeCatalog)
    }
}