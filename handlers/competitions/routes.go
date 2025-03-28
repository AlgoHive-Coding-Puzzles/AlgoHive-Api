package competitions

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to competitions
// r: the RouterGroup to which the routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	// Public routes
	r.GET("/competitions/user", middleware.SetUserIdMiddleware(), GetUserCompetitions)

	competitions := r.Group("/competitions")
	competitions.Use(middleware.AuthMiddleware())
	{
		// Competition management routes
		competitions.GET("/", GetAllCompetitions)
		competitions.GET("/:id", GetCompetition)
		competitions.POST("/", CreateCompetition)
		competitions.PUT("/:id", UpdateCompetition)
		competitions.PUT("/:id/finish", FinishCompetition)
		competitions.PUT("/:id/visibility", ToggleCompetitionVisibility)
		competitions.DELETE("/:id", DeleteCompetition)
		
		 // Competition group management routes
		competitions.GET("/:id/groups", GetCompetitionGroups)
		competitions.POST("/:id/groups/:group_id", AddGroupToCompetition)
		competitions.DELETE("/:id/groups/:group_id", RemoveGroupFromCompetition)
		
		 // Try management routes
		competitions.GET("/:id/tries", GetCompetitionTries)
		competitions.GET("/:id/users/:user_id/tries", GetUserCompetitionTries)
		competitions.POST("/input", GetInputFromCompetition)
		competitions.POST("/answer_puzzle", AnswerPuzzle)
		competitions.GET("/:id/puzzles/:puzzle_id/:puzzle_index/tries", GetTriesFromCompetitonPuzzle)
		competitions.GET("/:id/permission/puzzles/:puzzle_index", UserHasPermissionToViewPuzzle)
		competitions.GET("/:id/export", ExportCompetitionDataExcel)

		 // Statistics routes
		competitions.GET("/:id/statistics", GetCompetitionStatistics)
	}
}
