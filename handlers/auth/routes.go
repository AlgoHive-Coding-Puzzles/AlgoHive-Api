package auth

import (
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
		auth.POST("/logout", Logout)
		auth.POST("/request-reset", RequestPasswordReset)
		auth.POST("/reset-password", ResetPassword)
	}
}
