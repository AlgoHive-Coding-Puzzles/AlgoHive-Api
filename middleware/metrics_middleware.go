package middleware

import (
	"api/metrics"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware collects HTTP request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		// Increment in-progress counter
		metrics.RequestInProgress.WithLabelValues(method, path).Inc()
		
		// Start timer
		startTime := time.Now()
		
		// Process request
		c.Next()
		
		// Record request duration
		duration := time.Since(startTime).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		
		// Increment total requests counter
		metrics.RequestCounter.WithLabelValues(status, method, path).Inc()
		
		// Observe request duration
		metrics.RequestDuration.WithLabelValues(status, method, path).Observe(duration)
		
		// Decrement in-progress counter
		metrics.RequestInProgress.WithLabelValues(method, path).Dec()
	}
}

// UpdateSystemMetrics periodically updates system metrics
func UpdateSystemMetrics() {
	go func() {
		for {
			// Update memory stats
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			
			metrics.MemoryStats.WithLabelValues("alloc").Set(float64(memStats.Alloc))
			metrics.MemoryStats.WithLabelValues("sys").Set(float64(memStats.Sys))
			metrics.MemoryStats.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
			metrics.MemoryStats.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
			metrics.MemoryStats.WithLabelValues("heap_idle").Set(float64(memStats.HeapIdle))
			metrics.MemoryStats.WithLabelValues("heap_inuse").Set(float64(memStats.HeapInuse))
			
			// Update goroutine count
			metrics.GoroutineCount.Set(float64(runtime.NumGoroutine()))
			
			// Wait before next update
			time.Sleep(15 * time.Second)
		}
	}()
}
