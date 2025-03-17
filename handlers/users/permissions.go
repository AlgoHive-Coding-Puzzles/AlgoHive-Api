package users

import (
	"api/database"
	"api/models"
	"api/utils/permissions"
)

// UserOwnsTargetGroups checks if the authenticated user owns at least one group
// to which the target user belongs
// userID: ID of the authenticated user
// targetUserID: ID of the target user
// returns: true if the authenticated user owns at least one group of the target user
func UserOwnsTargetGroups(userID string, targetUserID string) bool {
    var count int64
    err := database.DB.Raw(`
        SELECT COUNT(DISTINCT g1.id) 
        FROM groups g1
        JOIN user_groups ug ON g1.id = ug.group_id
        WHERE ug.user_id = ? 
        AND g1.id IN (
            SELECT DISTINCT g2.id
            FROM groups g2
            JOIN scopes s ON g2.scope_id = s.id
            JOIN role_scopes rs ON s.id = rs.scope_id
            JOIN roles r ON rs.role_id = r.id
            JOIN user_roles ur ON r.id = ur.role_id
            WHERE ur.user_id = ?
        )
    `, targetUserID, userID).Count(&count).Error

    if err != nil {
        return false
    }
    
    return count > 0
}

// HasPermissionForUser checks if the user has the necessary permissions to act on the target user
// user: the authenticated user
// targetUserID: ID of the target user
// requiredPermission: required permission if the user is not the owner of the group
func HasPermissionForUser(user models.User, targetUserID string, requiredPermission int) bool {
    // Owners have all permissions
    if permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        return true
    }
    
    // Otherwise check if the user owns a group of the target user
    return UserOwnsTargetGroups(user.ID, targetUserID) || 
           permissions.RolesHavePermission(user.Roles, requiredPermission)
}
