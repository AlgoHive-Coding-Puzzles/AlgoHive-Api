package catalogs

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to catalogs
// r: the RouterGroup to which the routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	catalogs := r.Group("/catalogs")
	catalogs.Use(middleware.AuthMiddleware())
	{
		catalogs.GET("/", GetAllCatalogs)
		catalogs.GET("/:catalogID/themes", GetThemesFromCatalog)
		catalogs.GET("/:catalogID/themes/:themeID", GetThemeDetailsFromCatalog)
		catalogs.GET("/:catalogID/themes/:themeID/puzzles/:puzzleID", GetPuzzleFromThemeCatalog)
		catalogs.GET("/:catalogID/themes/:themeID/puzzles/:puzzleID/inputs/:inputID", GetPuzzleInputFromThemeCatalog)
		catalogs.GET("/:catalogID/themes/:themeID/puzzles/:puzzleID/inputs/:inputID/first/:solution", CheckFirstSolutionPuzzleUser)
		catalogs.GET("/:catalogID/themes/:themeID/puzzles/:puzzleID/inputs/:inputID/second/:solution", CheckSecondSolutionPuzzleUser)
	}
}
