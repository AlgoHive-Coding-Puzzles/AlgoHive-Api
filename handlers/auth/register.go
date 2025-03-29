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
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterUser handles registration of a new user
// @Summary User Register
// @Description Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "Registration Details"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /auth/register [post]
func RegisterUser(c *gin.Context) {
    // Add timeout context for database operations
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()
    
    var registerReq RegisterRequest
    
    if err := c.ShouldBindJSON(&registerReq); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }
    
    // Validate the password
    if err := utils.ValidatePassword(registerReq.Password); err != nil {
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
    
    // Check if the email already exists
    var existingUser models.User
    if err := tx.Where("email = ?", registerReq.Email).First(&existingUser).Error; err == nil {
        tx.Rollback()
        response.Error(c, http.StatusConflict, ErrEmailInUse)
        return
    }
    
    // Hash the password
    hashedPassword, err := utils.HashPassword(registerReq.Password)
    if err != nil {
        tx.Rollback()
        log.Printf("Failed to hash password during registration: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrHashPasswordFailed)
        return
    }
    
    // Create a new user
    user := models.User{
        Email:     registerReq.Email,
        Password:  hashedPassword,
        Firstname: registerReq.Firstname,
        Lastname:  registerReq.Lastname,
        Blocked:   false,
        HasDefaultPassword: false,
    }
    
    if err := tx.Create(&user).Error; err != nil {
        tx.Rollback()
        log.Printf("Failed to create user: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrUserCreateFailed)
        return
    }
    
    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        log.Printf("Failed to commit transaction: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrUserCreateFailed)
        return
    }
    
    // Generate a JWT token
    token, err := utils.GenerateJWT(user.ID, user.Email)
    if err != nil {
        log.Printf("Failed to generate JWT for new user: %v", err)
        response.Error(c, http.StatusInternalServerError, ErrTokenGenerateFailed)
        return
    }
    
    // Set the token as an HTTP-only cookie
    setCookieToken(c, token, false)
    
    // Fetch the user with roles and groups
    var completeUser models.User
    if err := database.DB.Preload("Roles").Preload("Groups").Where("id = ?", user.ID).First(&completeUser).Error; err != nil {
        log.Printf("Failed to fetch complete user data: %v", err)
        // Continue anyway since the user was created successfully
    } else {
        user = completeUser
    }
    
    log.Printf("New user registered: %s (%s)", user.Email, user.ID)
    
    c.JSON(http.StatusCreated, AuthResponse{
        Token:         token,
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