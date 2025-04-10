package v1

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// Register the endpoints for the v1 API
func Register(r *gin.Engine) {
    v1 := r.Group("/api/v1")

	// Add metrics middleware to all routes
	v1.Use(middleware.MetricsMiddleware())
	
	rateLimiter := middleware.NewRateLimiter(10000, 1500) // 100 requests per second, 150 burst
    v1.Use(middleware.RateLimiterMiddleware(rateLimiter))

	RegisterPingRoutes(v1)
	RegisterSupportRoutes(v1)
	RegisterAuthRoutes(v1)
	RegisterScopesRoutes(v1)
	RegisterApisRoutes(v1)
	RegisterUserRoutes(v1)
	RegisterGroupsRoutes(v1)
	RegisterRolesRoutes(v1)
	RegisterCompetitionsRoutes(v1)
	
	// Register metrics endpoint
	RegisterMetricsRoutes(v1)
}