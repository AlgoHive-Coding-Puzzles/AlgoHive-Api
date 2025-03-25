package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterMetricsRoutes registers routes for the metrics API
func RegisterMetricsRoutes(r *gin.RouterGroup) {
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}