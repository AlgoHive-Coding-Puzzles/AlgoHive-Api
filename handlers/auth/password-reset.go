package auth

import (
	"api/database"
	"api/models"
	"api/services"
	"api/utils"
	"api/utils/response"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RequestPasswordResetRequest struct {
    Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
    Token    string `json:"token" binding:"required"`
    Password string `json:"password" binding:"required,min=8"`
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
    var req RequestPasswordResetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    var user models.User
    if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            // Return success anyway to prevent email enumeration
            c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link will be sent"})
            return
        }
        response.Error(c, http.StatusInternalServerError, "Failed to process request")
        return
    }

    // Generate random token
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to generate reset token")
        return
    }
    token := hex.EncodeToString(b)

    // Delete any existing reset tokens for this user
    if err := database.DB.Where("user_id = ?", user.ID).Delete(&models.PasswordReset{}).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to process request")
        return
    }

    // Create new password reset entry
    resetEntry := models.PasswordReset{
        UserID: user.ID,
        Token:  token,
    }

    if err := database.DB.Create(&resetEntry).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to create reset token")
        return
    }

    // Send email
    emailService := services.NewEmailService()
    if err := emailService.SendPasswordResetEmail(user.Email, token); err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to send reset email")
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link will be sent"})
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
    var req ResetPasswordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Find reset entry
    var resetEntry models.PasswordReset
    if err := database.DB.Where("token = ?", req.Token).First(&resetEntry).Error; err != nil {
        response.Error(c, http.StatusBadRequest, "Invalid or expired reset token")
        return
    }

    // Check if token is not older than 1 hour
    createdAt := resetEntry.CreatedAt
    if time.Since(createdAt) > time.Hour {
        // Delete expired token
        database.DB.Delete(&resetEntry)
        response.Error(c, http.StatusBadRequest, "Reset token has expired")
        return
    }

    // Hash new password
    hashedPassword, err := utils.HashPassword(req.Password)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to process new password")
        return
    }

    if err := database.DB.Model(&models.User{}).Where("id = ?", resetEntry.UserID).
        Updates(models.User{Password: hashedPassword, HasDefaultPassword: false}).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to update password")
        return
    }

    // Delete used reset token
    database.DB.Delete(&resetEntry)

    c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}