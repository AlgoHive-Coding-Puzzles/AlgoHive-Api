package roles

// Constants for error messages
const (
	ErrRoleNotFound          = "Role not found"
	ErrUserNotFound          = "User not found"
	ErrNoPermissionCreate    = "User does not have permission to create roles"
	ErrNoPermissionView      = "User does not have permission to view all roles"
	ErrNoPermissionUpdate    = "User does not have permission to update roles"
	ErrNoPermissionDelete    = "User does not have permission to delete roles"
	ErrNoPermissionAttach    = "User does not have permission to attach roles to users"
	ErrNoPermissionDetach    = "User does not have permission to detach roles from users"
	ErrFailedRoleUserRemove  = "Failed to remove role associations from users"
	ErrFailedRoleScopeRemove = "Failed to remove role associations from scopes"
	ErrFailedRoleDelete      = "Failed to delete role"
	ErrFailedTxCommit        = "Failed to commit transaction"
)

// CreateRoleRequest defines the structure for creating a role
type CreateRoleRequest struct {
	Name       string   `json:"name" binding:"required"`
	Permission int      `json:"permission"`
	ScopesIds  []string `json:"scopes_ids"`
}

// UpdateRoleRequest defines the structure for updating a role
type UpdateRoleRequest struct {
	Name       string   `json:"name"`
	Permission int      `json:"permission"`
	ScopesIds  []string `json:"scopes_ids"`
}