<p align="center">
  <img width="150px" src="https://raw.githubusercontent.com/AlgoHive-Coding-Puzzles/Ressources/refs/heads/main/images/algohive-logo.png" title="Algohive">
</p>

<h1 align="center">AlgoHive API</h1>

<p align="center">
  <b>High-performance REST API for the AlgoHive coding challenge platform</b>
</p>

## Overview

AlgoHiveApi is a high-performance Go-based REST API backend for the AlgoHive coding challenge platform. Built with a focus on scalability, real-time capabilities, and comprehensive metrics, it provides the foundation for managing users, competitions, puzzles, and leaderboards in a secure and efficient manner.

## Technical Architecture

The AlgoHiveApi is designed with a clean, modular architecture:

- **Handler Layer**: Domain-specific route handlers (auth, competitions, users, etc.)
- **Service Layer**: Business logic implementation
- **Data Layer**: Database and cache interactions
- **Middleware Layer**: Cross-cutting concerns (auth, metrics, rate limiting)

The application follows RESTful design principles and includes real-time WebSocket capabilities for live updates.

## Key Features

- **High Performance**: Optimized request handling with concurrency support
- **Scalable Design**: Stateless architecture for horizontal scaling
- **Real-Time Updates**: WebSocket integration for live leaderboards and competition updates
- **Comprehensive Metrics**: Prometheus integration for detailed performance monitoring
- **Advanced Caching**: Redis-based caching for frequently accessed data
- **Rate Limiting**: Configurable rate limiting to prevent abuse
- **Secure Authentication**: JWT-based authentication with role-based access control
- **Comprehensive Documentation**: Swagger/OpenAPI documentation for all endpoints

## Performance Features

### Request Processing Optimization

- **Concurrent Request Handling**: Leverages Go's goroutines for efficient request processing
- **Database Connection Pooling**: Optimized database connection management
- **Context-Based Timeouts**: Prevents long-running operations from degrading performance

### Rate Limiting

```go
// Example rate limiting configuration
loginRateLimiter := middleware.NewRateLimiter(5, 10) // 5 req/s, 10 burst
catalogRateLimiter := middleware.NewRateLimiter(20, 20) // 20 req/s, 20 burst
```

## Cache Management

AlgoHiveApi implements a sophisticated caching strategy using Redis:

- **Competition Data Caching**: Frequently accessed competition details are cached
- **Leaderboard Caching**: Optimized leaderboard retrieval with automatic invalidation
- **Permission Caching**: User permissions are cached to reduce database lookups
- **Statistics Caching**: Competition statistics are cached with configurable TTL

Example cache implementation:

```go
// Cache competition statistics
statsJSON, err := utils.MarshalJSON(stats)
if err == nil {
    database.REDIS.Set(ctx, cacheKey, string(statsJSON), StatsCacheDuration)
}
```

## WebSocket Integration

Real-time updates are provided through WebSocket connections, particularly for competition leaderboards:

- **Client Registration**: WebSocket clients are registered per competition
- **Real-Time Updates**: Score changes are broadcasted to all connected clients
- **Connection Management**: Automatic connection cleanup and error handling
- **Scalable Architecture**: Designed to handle thousands of simultaneous connections

```go
// WebSocket endpoint for competition updates
r.GET("/competitions/:id/ws", competitions.CompetitionWebSocket)
```

## API Documentation

The API is fully documented using Swagger/OpenAPI:

- **Interactive Documentation**: Available at `/swagger/index.html`
- **Comprehensive Endpoints**: All endpoints are documented with parameters and responses
- **Authentication Details**: Clear documentation on authentication requirements
- **Response Schemas**: Detailed response structures

The API provides the following domain-specific endpoints:

- **Authentication**: User login, registration, token validation
- **Users**: User management, roles, and permissions
- **Competitions**: Creating, managing, and participating in coding competitions
- **Puzzles**: Accessing and submitting solutions to coding puzzles
- **Groups**: Managing user groups and access control
- **Scopes**: Managing access control scopes
- **Metrics**: System performance and utilization metrics

## Metrics and Monitoring

AlgoHiveApi includes comprehensive metrics collection via Prometheus:

- **HTTP Request Metrics**: Request counts, durations, and status codes
- **System Metrics**: CPU, memory, and disk utilization
- **Database Metrics**: Query performance and connection pool stats
- **Cache Metrics**: Hit rates and operation durations
- **Custom Business Metrics**: User activity, competition participation

```go
// Example metrics
metrics.RequestCounter.WithLabelValues(status, method, path).Inc()
metrics.RequestDuration.WithLabelValues(status, method, path).Observe(duration)
metrics.SystemCPUUsage.WithLabelValues(strconv.Itoa(i)).Set(percentage)
```

The metrics endpoint at `/api/v1/metrics` can be integrated with Grafana for advanced visualization and alerting.

## Installation

### Prerequisites

- Go 1.23 or higher
- PostgreSQL
- Redis
- Docker (for containerized deployment)

### Local Development

```bash
# Clone the repository
git clone https://github.com/AlgoHive-Coding-Puzzles/AlgoHiveApi.git
cd AlgoHiveApi

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Install dependencies
go mod tidy

# Run the server
go run main.go
```

### Docker Deployment

```bash
# Build the Docker image
docker build -t algohive-api .

# Run with Docker
docker run -d -p 8080:8080 \
  --name algohive-api \
  --env-file .env \
  algohive-api
```

## Development Commands

```bash
# Install dependencies
go mod tidy

# Run the development server with hot reload
air

# Update Swagger documentation
swag init

# Run tests
go test ./...

# Build for production
go build -o algohive-api
```

## High-Availability Deployment

For production environments, consider the following architecture:

- Multiple API instances behind a load balancer
- Replicated PostgreSQL database
- Redis cluster for distributed caching
- Prometheus and Grafana for monitoring
- Horizontal scaling based on load metrics

Example Docker Compose for high-availability setup is available in the `deployment` directory.

## Configuration

AlgoHiveApi can be configured via environment variables:

```bash
# Server configuration
API_PORT=8080
ENV=development
ALLOWED_ORIGINS=http://localhost:5173

# Database configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=algohive
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password

# Redis configuration
CACHE_HOST=localhost
CACHE_PORT=6379
CACHE_PASSWORD=
CACHE_DB=0
CACHE_EXPI_MIN=5

# Authentication
JWT_SECRET=your_secret_key
JWT_EXPIRATION=86400
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
