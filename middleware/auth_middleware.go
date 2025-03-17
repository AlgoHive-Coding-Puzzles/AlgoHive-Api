package middleware

import (
	"api/database"
	"api/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the JWT token and sets the user ID in the context
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try to get token from cookie first
        tokenCookie, err := c.Cookie("auth_token")
        
        // If no cookie, try to get from Authorization header as fallback
        if err != nil {
            authHeader := c.GetHeader("Authorization")
            if authHeader == "" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
                c.Abort()
                return
            }

            // Check if the header has the format "Bearer token"
            parts := strings.SplitN(authHeader, " ", 2)
            if !(len(parts) == 2 && parts[0] == "Bearer") {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
                c.Abort()
                return
            }
            
            tokenCookie = parts[1]
        }

        // Validate the token
        claims, err := utils.ValidateToken(tokenCookie)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // Check if token is in Redis blacklist (logged out tokens)
        redisKey := fmt.Sprintf("token:blacklist:%s", tokenCookie)
        ctx := c.Request.Context()
        exists, err := database.REDIS.Exists(ctx, redisKey).Result()
        if err == nil && exists > 0 {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been invalidated"})
            c.Abort()
            return
        }

        // Set user ID in context
        c.Set("userID", claims.UserID)
        c.Set("email", claims.Email)
        
        c.Next()
    }
}