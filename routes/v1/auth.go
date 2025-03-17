package v1

import (
	"api/handlers/auth"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers the routes for the v1 API of authentication
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterAuthRoutes(r *gin.RouterGroup) {
	auth.RegisterRoutes(r)
}