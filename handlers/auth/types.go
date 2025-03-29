package auth

import (
	"api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Constants for error messages
const (
	ErrInvalidCredentials  = "Invalid credentials"
	ErrAccountBlocked      = "Your account has been blocked"
	ErrEmailInUse          = "Email already in use"
	ErrHashPasswordFailed  = "Failed to hash password"
	ErrUserCreateFailed    = "Failed to create user"
	ErrUserLoginFailed	   = "Failed to login user"
	ErrTokenGenerateFailed = "Failed to generate token"
	ErrNoTokenProvided     = "No token provided"
	ErrFailedRequest       = "Failed to process request"
	ErrInvalidToken        = "Invalid token"
	ErrInvalidExpiredToken = "Invalid or expired token"
	ErrUserNotFound        = "User not found"
	ErrLogoutFailed        = "Failed to logout"
	ErrLogoutSuccess       = "Successfully logged out"
	ResetSuccessMsg        = "If the email exists, a reset link will be sent"
    ResetCompleteMsg       = "Password has been reset successfully"
)

const (
	// Token Password Reset constants
	TokenBytes       = 32
    TokenExpiration  = time.Hour
)

const (
    // Session constants
    TokenCookieName = "auth_token"
    AuthHeaderName  = "Authorization"
    AuthHeaderPrefix = "Bearer "
    SessionTimeout  = 5 * time.Second
    
    // Cache constants
    UserCacheDuration = 5 * time.Minute
    UserCacheKeyPrefix = "user_session:"
)

// LoginRequest model for login endpoints
type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// RegisterRequest model for registration
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Firstname string `json:"first_name" binding:"required"`
	Lastname  string `json:"last_name" binding:"required"`
	Password  string `json:"password" binding:"required,min=8"`
}

// AuthResponse model for authentication responses
type AuthResponse struct {
	Token         string        `json:"token"`
	UserID        string        `json:"user_id"`
	Email         string        `json:"email"`
	Firstname     string        `json:"first_name"`
	Lastname      string        `json:"last_name"`
	LastConnected *time.Time    `json:"last_connected"`
	Blocked 	  bool          `json:"blocked"`
	Permissions   int           `json:"permissions"`
	Roles         []models.Role  `json:"roles"`
	Groups        []models.Group `json:"groups"`
}

type RequestPasswordResetRequest struct {
    Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
    Token    string `json:"token" binding:"required"`
    Password string `json:"password" binding:"required,min=8"`
}

// setCookieToken sets the authentication token as a secure HTTP-only cookie
func setCookieToken(c *gin.Context, token string, rememberMe bool) {
	var maxAge time.Duration
	if rememberMe {
		maxAge = 30 * 24 * time.Hour // 30 days
	} else {
		maxAge = 1 * 24 * time.Hour // 1 day
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"auth_token",   // name
		token,          // value
		int(maxAge.Seconds()), // max age in seconds
		"/",            // path
		"",             // domain
		true,           // secure (HTTPS only)
		true,           // httpOnly (not accessible via JavaScript)
	)
}
