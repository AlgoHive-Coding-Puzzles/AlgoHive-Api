package scopes

// Constants for error messages
const (
	ErrScopeNotFound          = "Scope not found"
	ErrRoleNotFound           = "Role not found"
	ErrInvalidAPIEnvIDs       = "Invalid API environment IDs"
	ErrRolesRequired          = "Roles are required"
	ErrNoPermissionView       = "User does not have permission to view scopes"
	ErrNoPermissionCreate     = "User does not have permission to create scopes"
	ErrNoPermissionUpdate     = "User does not have permission to update scopes"
	ErrNoPermissionDelete     = "User does not have permission to delete scopes"
	ErrNoPermissionAttach     = "User does not have permission to attach scopes to roles"
	ErrNoPermissionDetach     = "User does not have permission to detach scopes from roles"
	ErrFailedCreateScope      = "Failed to create scope: "
	ErrFailedUpdateScope      = "Failed to update scope: "
	ErrFailedDeleteScope      = "Failed to delete scope: "
	ErrFailedAssociateAPIEnv  = "Failed to associate API environments: "
	ErrFailedUpdateAssoc      = "Failed to update scope associations: "
	ErrFailedClearAssoc       = "Failed to clear scope associations: "
	ErrFailedAttachRole       = "Failed to attach scope to role: "
	ErrFailedDetachRole       = "Failed to detach scope from role: "
	ErrFailedGetScopes        = "Failed to get scopes"
)

// CreateScopeRequest model for creating a new scope
type CreateScopeRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	CatalogsIds []string `json:"catalogs_ids" binding:"required"`
}