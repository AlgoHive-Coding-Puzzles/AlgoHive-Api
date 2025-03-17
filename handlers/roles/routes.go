package roles

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to role management
// r: the RouterGroup to which routes should be added
func RegisterRoutes(r *gin.RouterGroup) {    
    roles := r.Group("/roles")
    roles.Use(middleware.AuthMiddleware())
    {
		roles.POST("/", CreateRole)
		roles.GET("/", GetAllRoles)
		roles.GET("/:role_id", GetRoleByID)
		roles.PUT("/:role_id", UpdateRoleByID)
		roles.DELETE("/:role_id", DeleteRole)
        
        // Move user-role routes under /roles to avoid conflicts
        roles.POST("/attach/:role_id/to-user/:user_id", AttachRoleToUser)
        roles.DELETE("/detach/:role_id/from-user/:user_id", DetachRoleFromUser)
    }
}
