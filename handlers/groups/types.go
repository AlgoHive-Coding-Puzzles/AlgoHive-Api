package groups

// Constants for error messages
const (
	ErrGroupNotFound          = "Group not found"
	ErrUserNotFound           = "User not found"
	ErrScopeNotFound          = "Scope not found"
	ErrInvalidScopeIDs        = "Invalid scope IDs"
	ErrNoPermissionView       = "User does not have permission to view groups"
	ErrNoPermissionViewGroup  = "User does not have permission to view this group"
	ErrNoPermissionCreate     = "User does not have permission to create groups"
	ErrNoPermissionDelete     = "User does not have permission to delete this group"
	ErrNoPermissionUpdate     = "User does not have permission to update this group"
	ErrNoPermissionAddUser    = "User does not have permission to add users to this group"
	ErrNoPermissionRemoveUser = "User does not have permission to remove users from this group"
	ErrFetchingGroups         = "Error while fetching groups"
)

// CreateGroupRequest model for creating a group
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ScopeId     string `json:"scope_id" binding:"required"`
}

// UpdateGroupRequest model for updating a group
type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}