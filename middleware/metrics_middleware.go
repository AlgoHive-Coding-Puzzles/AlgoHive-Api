package middleware

import (
	"api/metrics"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
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
			
			// Memory metrics
			metrics.MemoryStats.WithLabelValues("alloc").Set(float64(memStats.Alloc))
			metrics.MemoryStats.WithLabelValues("sys").Set(float64(memStats.Sys))
			metrics.MemoryStats.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
			metrics.MemoryStats.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
			metrics.MemoryStats.WithLabelValues("heap_idle").Set(float64(memStats.HeapIdle))
			metrics.MemoryStats.WithLabelValues("heap_inuse").Set(float64(memStats.HeapInuse))
			
			 // CPU usage metrics
            updateCPUMetrics()

            // Disk usage metrics
            updateDiskMetrics()

            // System load metrics
            updateLoadMetrics()

			// Update goroutine count
			metrics.GoroutineCount.Set(float64(runtime.NumGoroutine()))
			
			// Wait before next update
			time.Sleep(15 * time.Second)
		}
	}()
}

func updateCPUMetrics() {
    percentages, err := cpu.Percent(0, true)
    if err != nil {
        return
    }
    
    for i, percentage := range percentages {
        metrics.SystemCPUUsage.WithLabelValues(strconv.Itoa(i)).Set(percentage)
    }
}

func updateDiskMetrics() {
    partitions, err := disk.Partitions(false)
    if err != nil {
        return
    }

    for _, partition := range partitions {
        usage, err := disk.Usage(partition.Mountpoint)
        if err != nil {
            continue
        }

        metrics.SystemDiskUsage.WithLabelValues(
            partition.Device,
            partition.Mountpoint,
            "used",
        ).Set(float64(usage.Used))

        metrics.SystemDiskUsage.WithLabelValues(
            partition.Device,
            partition.Mountpoint,
            "free",
        ).Set(float64(usage.Free))

        metrics.SystemDiskUsage.WithLabelValues(
            partition.Device,
            partition.Mountpoint,
            "total",
        ).Set(float64(usage.Total))
    }
}

func updateLoadMetrics() {
    loadStats, err := load.Avg()
    if err != nil {
        return
    }

    metrics.SystemLoadAverage.WithLabelValues("1min").Set(loadStats.Load1)
    metrics.SystemLoadAverage.WithLabelValues("5min").Set(loadStats.Load5)
    metrics.SystemLoadAverage.WithLabelValues("15min").Set(loadStats.Load15)
}
