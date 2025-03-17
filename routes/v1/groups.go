package v1

import (
	"api/handlers/groups"

	"github.com/gin-gonic/gin"
)

// RegisterGroupsRoutes registers the routes for the v1 API of groups
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterGroupsRoutes(r *gin.RouterGroup) {
	groups.RegisterRoutes(r)
}