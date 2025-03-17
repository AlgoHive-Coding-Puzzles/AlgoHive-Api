package catalogs

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllCatalogs récupère tous les catalogues
// @Summary Get all Catalogs Catalog
// @Description Get all Catalogs, only accessible to users with the API_ENV permission
// @Tags Catalogs
// @Accept json
// @Produce json
// @Success 200 {array} models.Catalog
// @Failure 401 {object} map[string]string
// @Router /catalogs [get]
// @Security Bearer
func GetAllCatalogs(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}
	
	var catalogs []models.Catalog
	if permissions.RolesHavePermission(user.Roles, permissions.API_ENV) || permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		if err := database.DB.Preload("Scopes").Find(&catalogs).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to fetch catalogs")
			return
		}
	} else {
		if err := database.DB.Raw(`
			SELECT DISTINCT c.*
			FROM public.catalogs c
			JOIN public.scope_catalogs sae ON sae.catalog_id = c.id
			JOIN public.role_scopes rs ON rs.scope_id = sae.scope_id
			JOIN public.user_roles ur ON ur.role_id = rs.role_id
			WHERE ur.user_id = ?`, user.ID).Scan(&catalogs).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to fetch catalogs")
			return
		}
	}
	
	c.JSON(http.StatusOK, catalogs)
}
