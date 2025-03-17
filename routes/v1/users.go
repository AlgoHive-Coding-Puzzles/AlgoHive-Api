package v1

import (
	"api/handlers/users"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers the endpoints for the v1 API of users
// This function now acts as a simple proxy to the dedicated handlers package
func RegisterUserRoutes(r *gin.RouterGroup) {
	users.RegisterRoutes(r)
}