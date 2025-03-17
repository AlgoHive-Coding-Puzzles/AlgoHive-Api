package groups

import (
	"api/database"
	"api/models"
	"api/utils/permissions"
)

// userCanManageGroup check if the user can manage the group
// userID: User ID
// group: Group object
// return: true if the user can manage the group
//         false if the user cannot manage the group
func userCanManageGroup(userID string, group *models.Group) bool {
	// Fetch in cascade the user with roles and scopes
	var user models.User
	if err := database.DB.Where("id = ?", userID).Preload("Roles.Scopes.Groups").First(&user).Error; err != nil {
		return false
	}
	
	for _, role := range user.Roles {
		for _, scope := range role.Scopes {
			for _, g := range scope.Groups {
				if g.ID == group.ID {
					return true
				}
			}
		}
	}
	
	return false
}

// UserOwnsTargetGroups check if the user owns the target groups
// userID: User ID
// targetID: Target ID
// return: true if the user owns the target groups
//         false if the user does not own the target groups
//         false if there is an error
func UserOwnsTargetGroups(userID string, targetID string) bool {
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
    `, targetID, userID).Count(&count).Error

    if err != nil {
        return false
    }
    
    return count > 0
}

// checkGroupPermission check if the user has permission to manage the group
// userID: User ID
// groupID: Group ID
// return: Group object and a boolean indicating if the user can manage the group
func checkGroupPermission(userID string, groupID string) (*models.Group, bool) {
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return nil, false
	}
	
	return &group, userCanManageGroup(userID, &group)
}

// hasGroupPermission check if the user has the required permission
// user: User object
// permission: Required permission
// return: true if the user has the permission
func hasGroupPermission(user models.User, permission int) bool {
	return permissions.IsStaff(user) || permissions.RolesHavePermission(user.Roles, permission)
}
