package roles

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"api/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateRole creates a new role
// @Summary Create a new Role
// @Description Create a new Role
// @Tags Roles
// @Accept json
// @Produce json
// @Param role body RoleRequest true "Role Profile"
// @Success 201 {object} models.Role
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /roles [post]
// @Security Bearer
func CreateRole(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionCreate)
        return
    }

    var createRoleRequest RoleRequest
    if err := c.ShouldBindJSON(&createRoleRequest); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Create the role with basic information
    role := models.Role{
        Name:        createRoleRequest.Name,
        Permissions: createRoleRequest.Permission,
    }

    // Start a transaction to ensure atomicity
    tx := database.DB.Begin()

    // Create the role first
    if err := tx.Create(&role).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to create role")
        return
    }

    // If scopes are specified, attach them to the role
    if len(createRoleRequest.ScopesIDs) > 0 {
        var scopes []models.Scope
        if err := tx.Where("id IN ?", createRoleRequest.ScopesIDs).Find(&scopes).Error; err != nil || len(scopes) != len(createRoleRequest.ScopesIDs) {
            tx.Rollback()
            response.Error(c, http.StatusBadRequest, "Invalid scope IDs")
            return
        }

        // Associate scopes with the role
        for i := range scopes {
            if err := tx.Model(&role).Association("Scopes").Append(&scopes[i]); err != nil {
                tx.Rollback()
                response.Error(c, http.StatusInternalServerError, "Failed to associate scopes with role")
                return
            }
        }
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Fetch the complete role with associations for response
    if err := database.DB.Preload("Scopes").First(&role, "id = ?", role.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to retrieve created role")
        return
    }

    c.JSON(http.StatusCreated, role)
}

// GetAllRoles retrieves all roles
// @Summary Get all Roles
// @Description Get all Roles
// @Tags Roles
// @Accept json
// @Produce json
// @Success 200 {array} models.Role
// @Failure 401 {object} map[string]string
// @Router /roles [get]
// @Security Bearer
func GetAllRoles(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    var roles []models.Role
    if err := database.DB.Preload("Scopes").Find(&roles).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to fetch roles")
        return
    }
    
    c.JSON(http.StatusOK, roles)
}

// GetRoleByID retrieves a role by its ID
// @Summary Get a Role by ID
// @Description Get a Role by ID
// @Tags Roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.Role
// @Failure 404 {object} map[string]string
// @Router /roles/{role_id} [get]
// @Security Bearer
func GetRoleByID(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionView)
        return
    }

    roleID := c.Param("role_id")

    var role models.Role
    if err := database.DB.Preload("Users").Preload("Scopes").First(&role, "id = ?",roleID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    c.JSON(http.StatusOK, role)
}

// UpdateRoleByID updates a role by its ID
// @Summary Update a Role by ID
// @Description Update a Role by ID
// @Tags Roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Param role body RoleRequest true "Role Update Data"
// @Success 200 {object} models.Role
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /roles/{role_id} [put]
// @Security Bearer
func UpdateRoleByID(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
        return
    }

    roleID := c.Param("role_id")

    var role models.Role
    if err := database.DB.First(&role, "id = ?", roleID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    // Parse the update request
    var updateRequest RoleRequest
    if err := c.ShouldBindJSON(&updateRequest); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Start a transaction
    tx := database.DB.Begin()

    // Update basic fields
    if err := tx.Model(&role).Updates(models.Role{
        Name:        updateRequest.Name,
        Permissions: updateRequest.Permission,
    }).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to update role")
        return
    }

    // If scopes are specified, update the scopes relationship
    if updateRequest.ScopesIDs != nil {
        // Clear existing associations
        if err := tx.Model(&role).Association("Scopes").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to clear existing scope associations")
            return
        }

        // Add new scope associations if any scope IDs are provided
        if len(updateRequest.ScopesIDs) > 0 {
            var scopes []models.Scope
            if err := tx.Where("id IN ?", updateRequest.ScopesIDs).Find(&scopes).Error; err != nil {
                tx.Rollback()
                response.Error(c, http.StatusBadRequest, "Invalid scope IDs")
                return
            }

            // Check if all requested scopes were found
            if len(scopes) != len(updateRequest.ScopesIDs) {
                tx.Rollback()
                response.Error(c, http.StatusBadRequest, "One or more scope IDs are invalid")
                return
            }

            // Associate scopes with the role
            for i := range scopes {
                if err := tx.Model(&role).Association("Scopes").Append(&scopes[i]); err != nil {
                    tx.Rollback()
                    response.Error(c, http.StatusInternalServerError, "Failed to associate scopes with role")
                    return
                }
            }
        }
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Fetch the updated role with all its associations for the response
    if err := database.DB.Preload("Users").Preload("Scopes").First(&role, "id = ?", roleID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to fetch updated role")
        return
    }

    c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role and its associations
// @Summary Delete a role
// @Description Delete a role and cascade first to roles_scopes and user_roles
// @Tags Roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /roles/{role_id} [delete]
// @Security Bearer
func DeleteRole(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
        response.Error(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

    roleID := c.Param("role_id")

    var role models.Role
    if err := database.DB.First(&role, "id = ?", roleID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    // Start a transaction to ensure atomicity of operations
    tx := database.DB.Begin()

    // Remove the role from all users who have it (user_roles)
    if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", roleID).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedRoleUserRemove)
        return
    }

    // Remove all associations with scopes (role_scopes)
    if err := tx.Exec("DELETE FROM role_scopes WHERE role_id = ?", roleID).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedRoleScopeRemove)
        return
    }

    // Finally, delete the role itself
    if err := tx.Delete(&role).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedRoleDelete)
        return
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, ErrFailedTxCommit)
        return
    }

    c.Status(http.StatusNoContent)
}