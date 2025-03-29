package auth

import (
	"api/database"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// getTokenFromRequest retrieves the token from the cookie or Authorization header
// This is optimized to reduce allocation and avoid string manipulation when possible
func getTokenFromRequest(c *gin.Context) (string, error) {
    // First try to get from cookie (fastest path)
    if token, err := c.Cookie(TokenCookieName); err == nil && token != "" {
        return token, nil
    }

    // Fall back to Authorization header if needed
    authHeader := c.GetHeader(AuthHeaderName)
    if authHeader == "" {
        return "", http.ErrNoCookie
    }

    // Efficiently check for Bearer prefix
    if !strings.HasPrefix(authHeader, AuthHeaderPrefix) {
        return "", http.ErrNoCookie
    }

    // Extract token without string concatenation
    return authHeader[len(AuthHeaderPrefix):], nil
}

// Logout handles user logout by invalidating their token
// @Summary User Logout
// @Description Logout a user by invalidating their token
// @Tags Auth
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/logout [post]
// @Security Bearer
func Logout(c *gin.Context) {
    // Use a timeout context to ensure the function completes in reasonable time
    ctx, cancel := context.WithTimeout(c.Request.Context(), SessionTimeout)
    defer cancel()
    
    // Retrieve the token
    token, err := getTokenFromRequest(c)
    if err != nil {
        response.Error(c, http.StatusUnauthorized, ErrNoTokenProvided)
        return
    }

    // Validate the token to obtain the expiration time
    claims, err := utils.ValidateToken(token)
    if err != nil {
        response.Error(c, http.StatusUnauthorized, ErrInvalidToken)
        return
    }

    // Calculate the remaining time until token expiration
    expirationTime := claims.ExpiresAt.Time
    remainingTime := time.Until(expirationTime)
    if remainingTime <= 0 {
        // Token already expired, just clear cookie and return success
        c.SetCookie(TokenCookieName, "", -1, "/", "", true, true)
        c.JSON(http.StatusOK, gin.H{"message": ErrLogoutSuccess})
        return
    }

    // Add to Redis blacklist with our timeout context
    redisKey := "token:blacklist:" + token
    err = database.REDIS.Set(ctx, redisKey, "1", remainingTime).Err()
    if err != nil {
        log.Printf("Error blacklisting token: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrLogoutFailed)
        return
    }

    // Also invalidate any user cache entries
    if claims.UserID != "" {
        cacheKey := UserCacheKeyPrefix + claims.UserID
        if err := database.REDIS.Del(ctx, cacheKey).Err(); err != nil {
            // Just log this error, don't fail the whole request
            log.Printf("Error clearing user cache: %v", err)
        }
    }

    // Clear the authentication cookie
    c.SetCookie(TokenCookieName, "", -1, "/", "", true, true)
    c.JSON(http.StatusOK, gin.H{"message": ErrLogoutSuccess})
}

// CheckAuth checks if the session token is still valid and returns user data
// @Summary Check if the sent cookie token session is still valid and return user data
// @Description Check if the sent cookie token session is still valid and return user data
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} AuthResponse
// @Failure 401 {object} map[string]string
// @Router /auth/check [get]
// @Security Bearer
func CheckAuth(c *gin.Context) {
    // Use a timeout context to ensure the function completes in reasonable time
    ctx, cancel := context.WithTimeout(c.Request.Context(), SessionTimeout)
    defer cancel()
    
    // Retrieve the token
    token, err := getTokenFromRequest(c)
    if err != nil {
        c.JSON(http.StatusOK, gin.H{"valid": false})
        return
    }

    // Validate the token
    claims, err := utils.ValidateToken(token)
    if err != nil {
        c.JSON(http.StatusOK, gin.H{"valid": false})
        return
    }
    
    // Check if token is blacklisted
    redisKey := "token:blacklist:" + token
    exists, err := database.REDIS.Exists(ctx, redisKey).Result()
    if err != nil {
        log.Printf("Error checking token blacklist: %v", err)
        // Continue anyway, better to check user than fail completely
    } else if exists == 1 {
        c.JSON(http.StatusOK, gin.H{"valid": false})
        return
    }
    
    // Try to get cached user data first
    cacheKey := UserCacheKeyPrefix + claims.UserID
    cachedData, err := database.REDIS.Get(ctx, cacheKey).Result()
    
    // If we have valid cached data, use it
    if err == nil && cachedData != "" {
        var response gin.H
        if err := utils.UnmarshalJSON([]byte(cachedData), &response); err == nil {
            c.JSON(http.StatusOK, response)
            return
        }
        // If unmarshalling fails, just continue to database lookup
    }

    // Retrieve user data from database
    var user models.User
    if err := database.DB.WithContext(ctx).
        Where("id = ?", claims.UserID).
        Preload("Roles").
        Preload("Groups").
        First(&user).Error; err != nil {
        c.JSON(http.StatusOK, gin.H{"valid": false})
        return
    }
    
    // Check if user is blocked
    if user.Blocked {
        c.JSON(http.StatusOK, gin.H{"valid": false})
        return
    }
    
    // Prepare response
    response := gin.H{
        "valid":             true,
        "hasDefaultPassword": user.HasDefaultPassword,
        "user": AuthResponse{
            UserID:        user.ID,
            Email:         user.Email,
            Firstname:     user.Firstname,
            Lastname:      user.Lastname,
            LastConnected: user.LastConnected,
            Blocked:       user.Blocked,
            Permissions:   permissions.MergeRolePermissions(user.Roles),
            Roles:         utils.ConvertRoles(user.Roles),
            Groups:        utils.ConvertGroups(user.Groups),
        },
    }
    
    // Cache the response for future requests
    responseBytes, err := utils.MarshalJSON(response)
    if err == nil {
        // Use shorter of token expiration or cache duration
        expiration := time.Until(claims.ExpiresAt.Time)
        if expiration > UserCacheDuration {
            expiration = UserCacheDuration
        }
        
        if err := database.REDIS.Set(ctx, cacheKey, string(responseBytes), expiration).Err(); err != nil {
            log.Printf("Failed to cache user session: %v", err)
            // Continue anyway, this is not critical
        }
    }
    
    c.JSON(http.StatusOK, response)
}