package v1

import (
	"api/handlers/catalogs"

	"github.com/gin-gonic/gin"
)

// RegisterApisRoutes registers the routes for the v1 API of catalogs
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterApisRoutes(r *gin.RouterGroup) {
	catalogs.RegisterRoutes(r)
}