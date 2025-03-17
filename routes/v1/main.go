package v1

import (
	"github.com/gin-gonic/gin"
)

// Register the endpoints for the v1 API
func Register(r *gin.Engine) {
    v1 := r.Group("/api/v1")

	RegisterPingRoutes(v1)
	RegisterAuthRoutes(v1)
	RegisterScopesRoutes(v1)
	RegisterApisRoutes(v1)
	RegisterUserRoutes(v1)
	RegisterGroupsRoutes(v1)
	RegisterRolesRoutes(v1)
	RegisterCompetitionsRoutes(v1)
}