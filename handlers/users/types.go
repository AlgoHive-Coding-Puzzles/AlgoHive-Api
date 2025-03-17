package users

import (
	"github.com/gin-gonic/gin"
)

// Error messages constants
const (
	ErrNotFound               = "User not found"
	ErrUserNotFound		      = "User not found"
	ErrGroupNotFound          = "Group not found"
	ErrRoleNotFound           = "Role not found"
	ErrUnauthorized           = "Unauthorized access"
	ErrFailedToHashPassword   = "Failed to hash password"
	ErrFailedToGetUsers       = "Failed to get users"
	ErrFailedToDeleteUser     = "Failed to delete user"
	ErrRolesRequired          = "Roles are required"
	ErrNoPermissionGroups     = "User does not have permission to create users with groups"
	ErrNoPermissionRoles      = "User does not have permission to create users with roles"
	ErrNoPermissionDelete     = "User does not have permission to delete this user"
	ErrNoPermissionBlock      = "User does not have permission to toggle block status"
	ErrNoPermissionUsersRoles = "User does not have permission to get users from roles"
	ErrFailedAssociationRoles = "Failed to remove user role associations"
	ErrFailedAssociationGroups = "Failed to remove user group associations"
)

// UserWithRoles represents a user with associated roles for API requests
type UserWithRoles struct {
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
}

// UserWithGroup represents a user with associated groups for API requests
type UserWithGroup struct {
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	Email     string   `json:"email"`
	Group     []string `json:"groups"`
}

// UserIdWithRoles represents a user ID with associated rolesIDs for API requests
type UserIdWithRoles struct {
	UserId string   `json:"user_id"`
	Roles  []string `json:"roles"`
}


// PasswordUpdate represents a password update request
type PasswordUpdate struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// respondWithError sends a JSON response with an error message
func respondWithError(c *gin.Context, status int, message string) {
    c.JSON(status, gin.H{"error": message})
}
