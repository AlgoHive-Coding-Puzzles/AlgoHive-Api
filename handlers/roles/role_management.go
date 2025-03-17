package roles

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateRole creates a new role
// @Summary Create a new Role
// @Description Create a new Role
// @Tags Roles
// @Accept json
// @Produce json
// @Param role body CreateRoleRequest true "Role Profile"
// @Success 200 {object} models.Role
// @Failure 400 {object} map[string]string
// @Router /roles [post]
// @Security Bearer
func CreateRole(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionCreate)
		return
	}

	var createRoleRequest CreateRoleRequest
	if err := c.ShouldBindJSON(&createRoleRequest); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// If scopes are specified, retrieve them
	var scopePointers []*models.Scope
	if len(createRoleRequest.ScopesIds) > 0 {
		var scopes []models.Scope
		if err := database.DB.Where("id IN (?)", createRoleRequest.ScopesIds).Find(&scopes).Error; err != nil {
			respondWithError(c, http.StatusBadRequest, "Invalid scope IDs")
			return
		}
		
		for i := range scopes {
			scopePointers = append(scopePointers, &scopes[i])
		}
	}

	// Create the role
	role := models.Role{
		Name:        createRoleRequest.Name,
		Permissions: createRoleRequest.Permission,
		Scopes:      scopePointers,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to create role")
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
// @Router /roles [get]
// @Security Bearer
func GetAllRoles(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.RolesHavePermission(user.Roles, permissions.ROLES) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var roles []models.Role
	if err := database.DB.Preload("Users").Preload("Scopes").Find(&roles).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch roles")
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
	roleID := c.Param("role_id")

	var role models.Role
	if err := database.DB.Where("id = ?", roleID).Preload("Users").Preload("Scopes").First(&role).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
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
// @Param role body UpdateRoleRequest true "Role Update Data"
// @Success 200 {object} models.Role
// @Failure 400 {object} map[string]string
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
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
		return
	}

	roleID := c.Param("role_id")

	var role models.Role
	if err := database.DB.Where("id = ?", roleID).Preload("Scopes").First(&role).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
		return
	}

	// Parse the update request
	var updateRequest UpdateRoleRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Update basic fields
	role.Name = updateRequest.Name
	role.Permissions = updateRequest.Permission

	// If scopes are specified, update the scopes relationship
	if updateRequest.ScopesIds != nil {
		// Clear existing associations
		if err := database.DB.Model(&role).Association("Scopes").Clear(); err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to clear existing scope associations")
			return
		}

		// Add new scope associations if any scope IDs are provided
		if len(updateRequest.ScopesIds) > 0 {
			var scopes []models.Scope
			if err := database.DB.Where("id IN (?)", updateRequest.ScopesIds).Find(&scopes).Error; err != nil {
				respondWithError(c, http.StatusBadRequest, "Invalid scope IDs")
				return
			}

			var scopePointers []*models.Scope
			for i := range scopes {
				scopePointers = append(scopePointers, &scopes[i])
			}

			// Set the new scopes
			role.Scopes = scopePointers
		}
	}

	// Save the updated role with its scope relationships
	if err := database.DB.Save(&role).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to update role")
		return
	}

	// Fetch the updated role with all its associations for the response
	if err := database.DB.Preload("Users").Preload("Scopes").Where("id = ?", roleID).First(&role).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch updated role")
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role and its associations
// @Summary delete a role
// @Description delete a role and cascade first to roles_scopes and user_roles
// @Tags Roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.Role
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
        respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDelete)
        return
    }

	roleID := c.Param("role_id")

    var role models.Role
    if err := database.DB.Where("id = ?", roleID).First(&role).Error; err != nil {
        respondWithError(c, http.StatusNotFound, ErrRoleNotFound)
        return
    }

    // Start a transaction to ensure atomicity of operations
    tx := database.DB.Begin()

    // Remove the role from all users who have it (user_roles)
    if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", roleID).Error; err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedRoleUserRemove)
        return
    }

    // Remove all associations with scopes (role_scopes)
    if err := tx.Exec("DELETE FROM role_scopes WHERE role_id = ?", roleID).Error; err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedRoleScopeRemove)
        return
    }

    // Finally, delete the role itself
    if err := tx.Delete(&role).Error; err != nil {
        tx.Rollback()
        respondWithError(c, http.StatusInternalServerError, ErrFailedRoleDelete)
        return
    }

    // Commit the transaction
    if err := tx.Commit().Error; err != nil {
        respondWithError(c, http.StatusInternalServerError, ErrFailedTxCommit)
        return
    }

    c.JSON(http.StatusOK, role)
}
