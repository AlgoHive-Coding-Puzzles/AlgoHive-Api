package auth

import (
	"api/config"
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes related to authentication
// r: the RouterGroup to which routes are added
func RegisterRoutes(r *gin.RouterGroup) {
	// Create a rate limiters
    loginRateLimiter := middleware.NewRateLimiter(40, 20) // 40 requests per minute with burst capacity
	resetRateLimiter := middleware.NewRateLimiter(20, 5) // 20 requests per minute with burst capacity
	
	// Check if LAN mode is enabled via environment variable
    if config.LANMode {
        loginRateLimiter.EnableLANMode()
        resetRateLimiter.EnableLANMode()
    }
	
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
