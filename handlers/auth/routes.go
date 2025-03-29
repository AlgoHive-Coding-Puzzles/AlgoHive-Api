package auth

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to authentication
// r: the RouterGroup to which routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	// Create a rate limiters
    loginRateLimiter := middleware.NewRateLimiter(5, 10)
	resetRateLimiter := middleware.NewRateLimiter(3, 5)
	
	auth := r.Group("/auth")
	{
		auth.POST("/login", middleware.RateLimiterMiddleware(loginRateLimiter), Login)
		auth.GET("/check", CheckAuth)
		auth.POST("/register", RegisterUser)
		auth.POST("/logout", Logout)
		auth.POST("/request-reset", middleware.RateLimiterMiddleware(resetRateLimiter), RequestPasswordReset)
		auth.POST("/reset-password", ResetPassword)
	}
}
