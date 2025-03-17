package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateUserAndAttachGroup creates a user and attaches groups to it
// @Summary Create a user and attach one or more groups
// @Description Create a new user and attach one or more roles to it
// @Tags Users
// @Accept json
// @Produce json
// @Param UserWithGroup body UserWithGroup true "User Profile with Groups"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Router /user/groups [post]
// @Security Bearer
func CreateUserAndAttachGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Check permissions
	if !permissions.IsStaff(user) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, ErrNoPermissionGroups)
		return
	}

	var userWithGroups UserWithGroup
	if err := c.ShouldBindJSON(&userWithGroups); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check that groups exist
	var groups []models.Group
	if err := database.DB.Where("id IN (?)", userWithGroups.Group).Find(&groups).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrGroupNotFound)
		return
	}
	
	if len(groups) == 0 {
		respondWithError(c, http.StatusNotFound, ErrGroupNotFound)
		return
	}

	// Create the user
	targetUser, err := createUser(userWithGroups.FirstName, userWithGroups.LastName, userWithGroups.Email)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, ErrFailedToHashPassword)
		return
	}

	// Associate groups to the user
	for i := range groups {
		if err := database.DB.Model(targetUser).Association("Groups").Append(&groups[i]); err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to attach group to user")
			return
		}
	}

	c.JSON(http.StatusCreated, targetUser)
}

// CreateBulkUsersAndAttachGroup creates multiple users and attaches a group to them
// @Summary Create Bulk Users and attach a Group
// @Description Create multiple new users and attach a group to them
// @Tags Users
// @Accept json
// @Produce json
// @Param users body []models.User true "Users Profiles"
// @Param group_id path string true "Group ID"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Router /user/group/{group_id}/bulk [post]
// @Security Bearer
func CreateBulkUsersAndAttachGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")

	// Check permissions
	if !UserOwnsTargetGroups(user.ID, groupID) {
		respondWithError(c, http.StatusUnauthorized, "User does not have permission to create users")
		return
	}
	
	// Check that the group exists
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrGroupNotFound)
		return
	}
	
	// Retrieve users to be created
	var users []models.User
	if err := c.ShouldBindJSON(&users); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	
	// Create the users and associate the group
	for i := range users {
		users[i].Groups = append(users[i].Groups, &group)
		hashedPassword, err := utils.HashPassword(users[i].Password)
		if err != nil {
			respondWithError(c, http.StatusInternalServerError, ErrFailedToHashPassword)
			return
		}
		users[i].Password = hashedPassword
		if err := database.DB.Create(&users[i]).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to create users")
			return
		}
	}
	
	c.JSON(http.StatusCreated, users)
}
