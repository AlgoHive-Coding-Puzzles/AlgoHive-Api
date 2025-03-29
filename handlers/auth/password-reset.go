package auth

import (
	"api/database"
	"api/models"
	"api/services"
	"api/utils"
	"api/utils/response"
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// generateResetToken creates a secure random token for password reset
func generateResetToken() (string, error) {
    b := make([]byte, TokenBytes)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// createResetEntry saves a new password reset token in the database using a transaction
func createResetEntry(userID string, token string) error {
    // Start a transaction
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Delete any existing reset tokens for this user
    if err := tx.Where("user_id = ?", userID).Delete(&models.PasswordReset{}).Error; err != nil {
        tx.Rollback()
        return err
    }

    // Create new password reset entry with explicit expiration time
    resetEntry := models.PasswordReset{
        UserID:    userID,
        Token:     token,
        ExpiresAt: time.Now().Add(TokenExpiration),
    }

    if err := tx.Create(&resetEntry).Error; err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit().Error
}

// RequestPasswordReset initiates the password reset process
// @Summary Request Password Reset
// @Description Send a password reset link to the user's email
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RequestPasswordResetRequest true "Email Request"
// @Success 200 {object} map[string]string
// @Failure 400,404 {object} map[string]string
// @Router /auth/request-reset [post]
func RequestPasswordReset(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    var resetRequest RequestPasswordResetRequest
    if err := c.ShouldBindJSON(&resetRequest); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    var user models.User
    if err := database.DB.WithContext(ctx).Where("email = ?", resetRequest.Email).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            // Return success anyway to prevent email enumeration
            c.JSON(http.StatusOK, gin.H{"message": ResetSuccessMsg})
            return
        }
        response.Error(c, http.StatusInternalServerError, ErrUserNotFound)
        return
    }

    // Generate random token
    token, err := generateResetToken()
    if err != nil {
        response.Error(c, http.StatusInternalServerError, ErrTokenGenerateFailed)
        return
    }

    // Create reset entry in database
    if err := createResetEntry(user.ID, token); err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedRequest)
        return
    }

    // Send email
    emailService := services.NewEmailService()
    if err := emailService.SendPasswordResetEmail(user.Email, token); err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedRequest)
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": ResetSuccessMsg})
}

// ResetPassword handles the password reset
// @Summary Reset Password
// @Description Reset user password using the reset token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset Request"
// @Success 200 {object} map[string]string
// @Failure 400,404 {object} map[string]string
// @Router /auth/reset-password [post]
func ResetPassword(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    var resetRequest ResetPasswordRequest
    if err := c.ShouldBindJSON(&resetRequest); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Validate password
    if err := utils.ValidatePassword(resetRequest.Password); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Start a transaction
    tx := database.DB.WithContext(ctx).Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Find reset entry
    var resetEntry models.PasswordReset
    if err := tx.Where("token = ?", resetRequest.Token).First(&resetEntry).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusBadRequest, ErrInvalidExpiredToken)
        return
    }

    // Check if token is expired
    if time.Now().After(resetEntry.ExpiresAt) {
        tx.Delete(&resetEntry)
        tx.Commit()
        response.Error(c, http.StatusBadRequest, "Reset token has expired")
        return
    }

    // Hash new password
    hashedPassword, err := utils.HashPassword(resetRequest.Password)
    if err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrHashPasswordFailed)
        return
    }

    // Update user password
    if err := tx.Model(&models.User{}).Where("id = ?", resetEntry.UserID).
        Updates(models.User{Password: hashedPassword, HasDefaultPassword: false}).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update password")
        return
    }

    // Delete used reset token
    if err := tx.Delete(&resetEntry).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to complete password reset")
        return
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to complete password reset")
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": ResetCompleteMsg})
}