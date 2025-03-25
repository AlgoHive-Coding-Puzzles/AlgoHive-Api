package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestCounter counts HTTP requests by status code, method, and path
	RequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "algohive_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"status", "method", "path"},
	)

	// RequestDuration measures HTTP request duration
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "algohive_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status", "method", "path"},
	)

	// RequestInProgress counts HTTP requests currently being processed
	RequestInProgress = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "algohive_http_requests_in_progress",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "path"},
	)

	// RateLimiterRejections counts rejected requests due to rate limiting
	RateLimiterRejections = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "algohive_rate_limiter_rejections_total",
			Help: "Total number of requests rejected by rate limiter",
		},
		[]string{"ip"},
	)
	
	// DatabaseOperationDuration measures database operation duration
	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "algohive_db_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// MemoryStats tracks memory usage stats
	MemoryStats = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "algohive_memory_stats_bytes",
			Help: "Memory statistics in bytes",
		},
		[]string{"type"},
	)

	// GoroutineCount tracks the number of goroutines
	GoroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "algohive_goroutine_count",
			Help: "Number of goroutines",
		},
	)

	// CacheHits counts the number of cache hits
    CacheHits = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "algohive_cache_hits_total",
            Help: "Total number of cache hits",
        },
    )

    // CacheMisses counts the number of cache misses
    CacheMisses = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "algohive_cache_misses_total", 
            Help: "Total number of cache misses",
        },
    )

    // SystemCPUUsage tracks CPU usage percentage
    SystemCPUUsage = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "algohive_system_cpu_usage_percent",
            Help: "CPU usage percentage by core",
        },
        []string{"core"},
    )

    // SystemDiskUsage tracks disk usage
    SystemDiskUsage = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "algohive_system_disk_usage_bytes",
            Help: "Disk usage statistics in bytes",
        },
        []string{"device", "mountpoint", "type"}, // type can be "used", "free", "total"
    )

    // SystemLoadAverage tracks system load averages
    SystemLoadAverage = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "algohive_system_load_average",
            Help: "System load average",
        },
        []string{"period"}, // "1min", "5min", "15min"
    )
)

// RecordDBOperation records the duration of a database operation
func RecordDBOperation(operation string, table string, startTime time.Time) {
	duration := time.Since(startTime).Seconds()
	DatabaseOperationDuration.WithLabelValues(operation, table).Observe(duration)
}
