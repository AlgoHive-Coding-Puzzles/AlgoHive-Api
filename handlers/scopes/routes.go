package scopes

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to scopes
// r: the RouterGroup to which the routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	scopes := r.Group("/scopes")
	scopes.Use(middleware.AuthMiddleware())
	{
		// Basic CRUD routes
		scopes.POST("/", CreateScope)
		scopes.GET("/", GetAllScopes)
		scopes.GET("/:scope_id", GetScope)
		scopes.PUT("/:scope_id", UpdateScope)
		scopes.DELETE("/:scope_id", DeleteScope)
		
		 // User-specific routes
		scopes.GET("/user", GetUserScopes)
		
		// Routes for role relationships
		scopes.GET("/roles", GetRoleScopes)
		scopes.POST("/:scope_id/roles/:role_id", AttachScopeToRole)
		scopes.DELETE("/:scope_id/roles/:role_id", DetachScopeFromRole)
	}
}
