package auth

import (
	"api/database"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RefreshToken issues a new JWT for the current session and invalidates the
// previous one (rotation), so a leaked or stale token cannot be replayed.
// @Summary Refresh authentication token
// @Description Rotate the current JWT: issue a new token and blacklist the previous one
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} AuthResponse
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
// @Security Bearer
func RefreshToken(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), SessionTimeout)
	defer cancel()

	oldToken, err := getTokenFromRequest(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, ErrNoTokenProvided)
		return
	}

	claims, err := utils.ValidateToken(oldToken)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			response.Error(c, http.StatusUnauthorized, ErrInvalidExpiredToken)
			return
		}
		response.Error(c, http.StatusUnauthorized, ErrInvalidToken)
		return
	}

	// Reject tokens that were already invalidated (logout or a previous refresh)
	redisKey := "token:blacklist:" + oldToken
	if blacklisted, err := database.REDIS.Exists(ctx, redisKey).Result(); err == nil && blacklisted > 0 {
		response.Error(c, http.StatusUnauthorized, ErrInvalidToken)
		return
	}

	var user models.User
	if err := database.DB.WithContext(ctx).
		Where("id = ?", claims.UserID).
		Preload("Roles").Preload("Groups").
		First(&user).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, ErrUserNotFound)
		return
	}

	if user.Blocked {
		response.Error(c, http.StatusUnauthorized, ErrAccountBlocked)
		return
	}

	newToken, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrTokenGenerateFailed)
		return
	}

	// Rotate: invalidate the old token for the remainder of its original lifetime
	if remaining := time.Until(claims.ExpiresAt.Time); remaining > 0 {
		if err := database.REDIS.Set(ctx, redisKey, "1", remaining).Err(); err != nil {
			log.Printf("Error blacklisting old token during refresh: %v", err)
		}
	}

	setCookieToken(c, newToken, false)

	c.JSON(http.StatusOK, AuthResponse{
		Token:         newToken,
		UserID:        user.ID,
		Email:         user.Email,
		Firstname:     user.Firstname,
		Lastname:      user.Lastname,
		LastConnected: user.LastConnected,
		Blocked:       user.Blocked,
		Permissions:   permissions.MergeRolePermissions(user.Roles),
		Roles:         utils.ConvertRoles(user.Roles),
		Groups:        utils.ConvertGroups(user.Groups),
	})
}
