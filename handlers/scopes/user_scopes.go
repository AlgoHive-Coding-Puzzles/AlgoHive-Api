package scopes

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserScopes retrieves all scopes that the user has access to
// @Summary Get all scopes that the user has access to
// @Description Get all scopes that the user has access to (based on roles) if the user has the SCOPES permission we return all scopes
// @Tags Scopes
// @Accept json
// @Produce json
// @Success 200 {array} models.Scope
// @Failure 401 {object} map[string]string
// @Router /scopes/user [get]
// @Security Bearer
func GetUserScopes(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	var scopes []models.Scope
	
	// If the user has the SCOPES permission, return all scopes
	if permissions.RolesHavePermission(user.Roles, permissions.SCOPES) {
		if err := database.DB.Preload("Catalogs").Preload("Roles").Preload("Roles.Scopes").Preload("Roles.Scopes.Groups").Preload("Groups").Find(&scopes).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, ErrFailedGetScopes)
			return
		}
	} else {
			// Otherwise, retrieve only the scopes they have access to via their roles
		if err := database.DB.Model(&user).Preload("Roles").Preload("Roles.Scopes").Preload("Roles.Scopes.Groups").First(&user).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to load user data")
			return
		}
		
		for _, role := range user.Roles {
			for _, scope := range role.Scopes {
				if !utils.ContainsScope(scopes, scope.ID) {
					scopes = append(scopes, *scope)
				}
			}
		}
	}

	c.JSON(http.StatusOK, scopes)
}
