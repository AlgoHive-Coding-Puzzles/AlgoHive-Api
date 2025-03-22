package v1

import (
	"api/middleware"

	"github.com/gin-gonic/gin"
)

// Register the endpoints for the v1 API
func Register(r *gin.Engine) {
    v1 := r.Group("/api/v1")

	rateLimiter := middleware.NewRateLimiter(100, 100) // 100 requêtes par minute
    v1.Use(middleware.RateLimiterMiddleware(rateLimiter))

	RegisterPingRoutes(v1)
	RegisterAuthRoutes(v1)
	RegisterScopesRoutes(v1)
	RegisterApisRoutes(v1)
	RegisterUserRoutes(v1)
	RegisterGroupsRoutes(v1)
	RegisterRolesRoutes(v1)
	RegisterCompetitionsRoutes(v1)
}