package v1

import (
	"api/handlers/competitions"

	"github.com/gin-gonic/gin"
)

// RegisterCompetitionsRoutes registers routes for the competitions v1 API
// This function serves as a proxy to the dedicated handlers package
func RegisterCompetitionsRoutes(r *gin.RouterGroup) {
	competitions.RegisterRoutes(r)
}
