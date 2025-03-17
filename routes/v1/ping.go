package v1

import (
	"github.com/gin-gonic/gin"
)

// @Summary Répond avec "pong"
// @Description Répond avec "pong"
// @Tags App
// @Produce json
// @Success 200 {object} map[string]string
// @Router /ping [get]
func pong(c *gin.Context) {
    c.JSON(200, gin.H{
        "message": "pong",
    })
}

func RegisterPingRoutes(r *gin.RouterGroup) {
    r.GET("/ping", pong)
}