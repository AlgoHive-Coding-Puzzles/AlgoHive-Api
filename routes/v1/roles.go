package v1

import (
	"api/handlers/roles"

	"github.com/gin-gonic/gin"
)

// RegisterRolesRoutes registers the routes for API v1 roles
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterRolesRoutes(r *gin.RouterGroup) {    
    roles.RegisterRoutes(r)
}