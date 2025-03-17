package v1

import (
	"api/handlers/scopes"

	"github.com/gin-gonic/gin"
)

// RegisterScopesRoutes registers the routes for API v1 scopes
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterScopesRoutes(r *gin.RouterGroup) {
	scopes.RegisterRoutes(r)
}