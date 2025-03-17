package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// hasCompetitionPermission checks if the user has a specific permission
func hasCompetitionPermission(user models.User, permission int) bool {
	return permissions.IsStaff(user) || permissions.RolesHavePermission(user.Roles, permission)
}

// GetAllCompetitions retrieves all competitions
// @Summary Get all competitions
// @Description Get all competitions, only accessible to users with the COMPETITIONS permission
// @Tags Competitions
// @Accept json
// @Produce json
// @Success 200 {array} models.Competition
// @Failure 401 {object} map[string]string
// @Router /competitions [get]
// @Security Bearer
func GetAllCompetitions(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) && !permissions.IsOwner(user) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	var competitions []models.Competition
	if err := database.DB.Preload("Catalog").Preload("Groups").Find(&competitions).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedFetchCompetitions)
		return
	}

	c.JSON(http.StatusOK, competitions)
}

// GetUserCompetitions retrieves competitions accessible to the current user
// @Summary Get user accessible competitions
// @Description Get all competitions accessible to the current user through their groups
// @Tags Competitions
// @Accept json
// @Produce json
// @Success 200 {array} models.Competition
// @Failure 401 {object} map[string]string
// @Router /competitions/user [get]
// @Security Bearer
func GetUserCompetitions(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// If the user has the COMPETITIONS permission or is OWNER, then show all competitions
	if hasCompetitionPermission(user, permissions.COMPETITIONS) || permissions.IsOwner(user) {
		GetAllCompetitions(c)
		return
	}

	var competitions []models.Competition
	// Retrieve competitions accessible through the user's groups
	if err := database.DB.Raw(`
		SELECT DISTINCT c.*
		FROM competitions c
		JOIN competition_groups cat ON c.id = cat.competition_id
		JOIN user_groups ug ON cat.group_id = ug.group_id
		WHERE ug.user_id = ? AND c.show = true
	`, user.ID).Scan(&competitions).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedFetchCompetitions)
		return
	}

	// Preload associations
	for i := range competitions {
		database.DB.Preload("Catalog").Preload("Groups").First(&competitions[i], competitions[i].ID)
	}

	c.JSON(http.StatusOK, competitions)
}

// GetCompetition retrieves a competition by ID
// @Summary Get a competition by ID
// @Description Get a competition by ID
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {object} models.Competition
// @Failure 404 {object} map[string]string
// @Router /competitions/{id} [get]
// @Security Bearer
func GetCompetition(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	competitionID := c.Param("id")
	var competition models.Competition

	if err := database.DB.Preload("Catalog").Preload("Groups").Where("id = ?", competitionID).First(&competition).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Check if the user has access to this competition
	hasAccess := hasCompetitionPermission(user, permissions.COMPETITIONS) || 
				permissions.IsOwner(user) ||
				userHasAccessToCompetition(user.ID, competitionID)

	if !hasAccess {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	c.JSON(http.StatusOK, competition)
}

// CreateCompetition creates a new competition
// @Summary Create a competition
// @Description Create a new competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param competition body CreateCompetitionRequest true "Competition to create"
// @Success 201 {object} models.Competition
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /competitions [post]
// @Security Bearer
func CreateCompetition(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionCreate)
		return
	}

	var req CreateCompetitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Check that the API environment exists
	var apiEnv models.Catalog
	if err := database.DB.First(&apiEnv, "id = ?", req.CatalogID).Error; err != nil {
		respondWithError(c, http.StatusBadRequest, ErrCatalogNotFound)
		return
	}

	// Create the competition
	competition := models.Competition{
		Title:           req.Title,
		Description:     req.Description,
		CatalogTheme:        req.CatalogTheme,
		CatalogID: req.CatalogID,
		Finished:        false,
		Show:            req.Show,
	}

	// Transaction to ensure atomic operations
	tx := database.DB.Begin()

	if err := tx.Create(&competition).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedCreateCompetition)
		return
	}

	// Append groups if specified
	if len(req.GroupIds) > 0 {
		var groups []models.Group
		if err := database.DB.Where("id IN ?", req.GroupIds).Find(&groups).Error; err != nil {
			tx.Rollback()
			respondWithError(c, http.StatusBadRequest, ErrGroupNotFound)
			return
		}

		if err := tx.Model(&competition).Association("Groups").Append(groups); err != nil {
			tx.Rollback()
			respondWithError(c, http.StatusInternalServerError, ErrFailedAddGroup)
			return
		}
	}

	tx.Commit()

	// Reload the competition with associations
	database.DB.Preload("Catalog").Preload("Groups").Where("id = ?", competition.ID).First(&competition)

	c.JSON(http.StatusCreated, competition)
}

// UpdateCompetition updates an existing competition
// @Summary Update a competition
// @Description Update an existing competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Param competition body UpdateCompetitionRequest true "Updated competition data"
// @Success 200 {object} models.Competition
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id} [put]
// @Security Bearer
func UpdateCompetition(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionUpdate)
		return
	}

	competitionID := c.Param("id")
	var competition models.Competition
	if err := database.DB.Where("id = ?", competitionID).First(&competition).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	var req UpdateCompetitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Update provided fields
	updateData := make(map[string]interface{})
	
	if req.Title != "" {
		updateData["title"] = req.Title
	}
	if req.Description != "" {
		updateData["description"] = req.Description
	}
	if req.CatalogTheme != "" {
		updateData["catalog_theme"] = req.CatalogTheme
	}
	if req.CatalogID != "" {
		// Check that the API environment exists
		var apiEnv models.Catalog
		if err := database.DB.First(&apiEnv, "id = ?", req.CatalogID).Error; err != nil {
			respondWithError(c, http.StatusBadRequest, ErrCatalogNotFound)
			return
		}
		updateData["catalog_id"] = req.CatalogID
	}
	if req.Finished != nil {
		updateData["finished"] = *req.Finished
	}
	if req.Show != nil {
		updateData["show"] = *req.Show
	}

	if err := database.DB.Model(&competition).Updates(updateData).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedUpdateCompetition)
		return
	}

	// Reload the competition with associations
	database.DB.Preload("Catalog").Preload("Groups").Where("id = ?", competition.ID).First(&competition)

	c.JSON(http.StatusOK, competition)
}

// DeleteCompetition deletes a competition and all related data
// @Summary Delete a competition
// @Description Delete a competition and all related data
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 204 {object} string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id} [delete]
// @Security Bearer
func DeleteCompetition(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionDelete)
		return
	}

	competitionID := c.Param("id")
	var competition models.Competition
	if err := database.DB.Where("id = ?", competitionID).First(&competition).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Transaction to ensure atomic operations
	tx := database.DB.Begin()

	// First, delete tries related to this competition
	if err := tx.Where("competition_id = ?", competitionID).Delete(&models.Try{}).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedDeleteCompetition)
		return
	}

	// Remove associations with groups
	if err := tx.Exec("DELETE FROM competition_groups WHERE competition_id = ?", competitionID).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedDeleteCompetition)
		return
	}

	// Delete the competition
	if err := tx.Delete(&competition).Error; err != nil {
		tx.Rollback()
		respondWithError(c, http.StatusInternalServerError, ErrFailedDeleteCompetition)
		return
	}

	tx.Commit()

	c.Status(http.StatusNoContent)
}

// FinishCompetition toggle the finished status of a competition
// @Summary Finish a competition
// @Description Toggle the finished status of a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {object} models.Competition
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/finish [put]
// @Security Bearer
func FinishCompetition(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionFinish)
		return
	}

	competitionID := c.Param("id")
	var competition models.Competition
	if err := database.DB.Where("id = ?", competitionID).First(&competition).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Toggle the finished status
	competition.Finished = !competition.Finished

	if err := database.DB.Save(&competition).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedToggleFinished)
		return
	}

	c.JSON(http.StatusOK, competition)
}

// Toggle visibility of a competition
// @Summary Toggle visibility of a competition
// @Description Toggle visibility of a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param id path string true "Competition ID"
// @Success 200 {object} models.Competition
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /competitions/{id}/visibility [put]
// @Security Bearer
func ToggleCompetitionVisibility(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	if !hasCompetitionPermission(user, permissions.COMPETITIONS) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionView)
		return
	}

	competitionID := c.Param("id")
	var competition models.Competition
	if err := database.DB.Where("id = ?", competitionID).First(&competition).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrCompetitionNotFound)
		return
	}

	// Toggle the visibility status
	competition.Show = !competition.Show

	if err := database.DB.Save(&competition).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedToggleVisibility)
		return
	}

	c.JSON(http.StatusOK, competition)
}

// userHasAccessToCompetition checks if a user has access to a competition through their groups
func userHasAccessToCompetition(userID string, competitionID string) bool {
	var count int64
	err := database.DB.Raw(`
		SELECT COUNT(*)
		FROM competition_groups cat
		JOIN user_groups ug ON cat.group_id = ug.group_id
		WHERE ug.user_id = ? AND cat.competition_id = ? AND (
			SELECT show FROM competitions WHERE id = ?
		) = true
	`, userID, competitionID, competitionID).Count(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}
