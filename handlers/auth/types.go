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
	ErrTokenGenerateFailed = "Failed to generate token"
	ErrNoTokenProvided     = "No token provided"
	ErrInvalidToken        = "Invalid token"
	ErrInvalidExpiredToken = "Invalid or expired token"
	ErrUserNotFound        = "User not found"
	ErrLogoutFailed        = "Failed to logout"
	ErrLogoutSuccess       = "Successfully logged out"
)

// LoginRequest model for login endpoints
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest model for registration
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	Firstname string `json:"firstname" binding:"required"`
	Lastname  string `json:"lastname" binding:"required"`
}

// AuthResponse model for authentication responses
type AuthResponse struct {
	Token         string        `json:"token"`
	UserID        string        `json:"user_id"`
	Email         string        `json:"email"`
	Firstname     string        `json:"firstname"`
	Lastname      string        `json:"lastname"`
	LastConnected *time.Time    `json:"last_connected"`
	Blocked 	 bool          `json:"blocked"`
	Permissions   int           `json:"permissions"`
	Roles         []models.Role  `json:"roles"`
	Groups        []models.Group `json:"groups"`
}

// respondWithError sends a standardized error response
func respondWithError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// setCookieToken sets the authentication token as a secure HTTP-only cookie
func setCookieToken(c *gin.Context, token string) {
	// Set the expiration time to match the JWT validity (usually 24 hours)
	cookieMaxAge := 24 * 3600 // 24 hours in seconds

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"auth_token",   // name
		token,          // value
		cookieMaxAge,   // max age
		"/",            // path
		"",             // domain
		true,           // secure (HTTPS only)
		true,           // httpOnly (not accessible via JavaScript)
	)
}
