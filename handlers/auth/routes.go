package auth

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to authentication
// r: the RouterGroup to which routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	
	auth := r.Group("/auth")
	{
		auth.POST("/login", Login)
		auth.GET("/check", CheckAuth)
		auth.POST("/register", RegisterUser)
		auth.POST("/logout", middleware.AuthMiddleware(), Logout)
	}
}
