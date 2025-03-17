package competitions

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to competitions
// r: the RouterGroup to which the routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	competitions := r.Group("/competitions")
	competitions.Use(middleware.AuthMiddleware())
	{
		// Competition management routes
		competitions.GET("/", GetAllCompetitions)
		competitions.GET("/user", GetUserCompetitions)
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
		competitions.POST("/:id/tries", StartCompetitionTry)
		competitions.PUT("/:id/tries/:try_id", FinishCompetitionTry)
		competitions.GET("/:id/users/:user_id/tries", GetUserCompetitionTries)
		
		 // Statistics routes
		competitions.GET("/:id/statistics", GetCompetitionStatistics)
	}
}
