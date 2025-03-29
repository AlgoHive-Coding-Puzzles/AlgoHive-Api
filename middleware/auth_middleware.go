package middleware

import (
	"api/database"
	"api/metrics"
	"api/utils"
	"api/utils/response"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
    // ContextKeyUserID is the key for storing the user ID in context
    ContextKeyUserID = "userID"
    // ContextKeyEmail is the key for storing the email in context
    ContextKeyEmail = "email"
    // TokenBlacklistPrefix is the prefix for Redis token blacklist keys
    TokenBlacklistPrefix = "token:blacklist:"
    // RedisTimeout is the timeout for Redis operations
    RedisTimeout = 500 * time.Millisecond
)

// getTokenFromRequest extracts the token from either cookie or Authorization header
func getTokenFromRequest(c *gin.Context) (string, error) {
    // Try to get token from cookie first
    if tokenCookie, err := c.Cookie("auth_token"); err == nil {
        return tokenCookie, nil
    }
    
    // Fall back to Authorization header
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        return "", fmt.Errorf("no authentication token found")
    }

    // Check if the header has the format "Bearer token"
    parts := strings.SplitN(authHeader, " ", 2)
    if !(len(parts) == 2 && parts[0] == "Bearer") {
        return "", fmt.Errorf("invalid authorization header format")
    }
    
    return parts[1], nil
}

// isBlacklisted checks if a token is blacklisted in Redis
func isBlacklisted(ctx context.Context, token string) (bool, error) {
    startTime := time.Now()
    defer func() {
        metrics.RecordDBOperation("exists", "redis_token_blacklist", startTime)
    }()
    
    // Create a context with timeout for the Redis operation
    redisCtx, cancel := context.WithTimeout(ctx, RedisTimeout)
    defer cancel()
    
    redisKey := TokenBlacklistPrefix + token
    exists, err := database.REDIS.Exists(redisCtx, redisKey).Result()
    if err != nil {
        return false, err
    }
    
    return exists > 0, nil
}

// AuthMiddleware validates the JWT token and sets the user ID in the context
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        
        // Get the token from the request
        token, err := getTokenFromRequest(c)
        if err != nil {
            response.Error(c, http.StatusUnauthorized, "Authentication required")
            c.Abort()
            return
        }
        
        // Validate the token
        claims, err := utils.ValidateToken(token)
        if err != nil {
            response.Error(c, http.StatusUnauthorized, "Invalid or expired token")
            c.Abort()
            return
        }
        
        // Check if token is blacklisted
        blacklisted, err := isBlacklisted(c.Request.Context(), token)
        if err != nil {
            
        } else if blacklisted {
            response.Error(c, http.StatusUnauthorized, "Token has been invalidated")
            c.Abort()
            return
        }
        
        // Set user data in context
        c.Set(ContextKeyUserID, claims.UserID)
        
        c.Next()
    }
}

// OptionalAuthMiddleware sets the user ID in context if authenticated, otherwise sets to empty string
func OptionalAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Default - no authenticated user
        c.Set(ContextKeyUserID, "")
        
        // Try to get the token
        token, err := getTokenFromRequest(c)
        if err != nil {
            c.Next()
            return
        }
        
        // Validate the token
        claims, err := utils.ValidateToken(token)
        if err != nil {
            c.Next()
            return
        }
        
        // Check if token is blacklisted
        blacklisted, err := isBlacklisted(c.Request.Context(), token)
        if err != nil || blacklisted {
            c.Next()
            return
        }
        
        // Set the user ID in context
        c.Set(ContextKeyUserID, claims.UserID)
        
        c.Next()
    }
}