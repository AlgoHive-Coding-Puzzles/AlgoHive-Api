package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// UserGroup fetch the authenticated user's groups
// @Summary Get the authenticated user's groups
// @Description Get the authenticated user's groups
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {array} models.Group
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /user/groups [get]
// @Security Bearer
func GetUserGroups(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	var groups []models.Group
	if err := database.DB.Model(user).Association("Groups").Find(&groups); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to retrieve groups")
		return
	}

	c.JSON(http.StatusOK, groups)
}

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

// DeleteAllUsersFromGroup deletes all users from a group
// @Summary Delete all users from a group
// @Description Delete all users from a group (Owner permission required)
// @Tags Users
// @Param group_id path string true "Group ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/group/{group_id}/delete [delete]
// @Security Bearer
func DeleteAllUsersFromGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return
    }

    groupID := c.Param("group_id")

    // Check permissions
    if !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        respondWithError(c, http.StatusUnauthorized, "User does not have permission to delete users")
        return
    }

    // Check that the group exists
    var group models.Group
    if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
        respondWithError(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

	// Get all users in the group
	var users []models.User
	if err := database.DB.Model(&group).Association("Users").Find(&users); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to retrieve users from group")
		return
	}
	
	// Delete all the associated users from the group
	if err := database.DB.Model(&group).Association("Users").Delete(users); err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to delete users from group")
		return
	}

    // Delete all user from the users table
	for _, user := range users {
		if err := database.DB.Delete(&user).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete user %s %s: %s", user.Firstname, user.Lastname, err.Error()))
			return
		}
	}


    c.JSON(http.StatusOK, gin.H{"message": "All users removed from group"})
}

// ImportUsersFromXLSXToGroup imports users from an XLSX file and attaches them to a group
// @Summary Import users from XLSX file and attach to a group
// @Description Upload an XLSX file containing user data and create users with the specified group
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Param group_id path string true "Group ID"
// @Param file formData file true "XLSX file containing user data"
// @Success 201 {array} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/group/{group_id}/import [post]
// @Security Bearer
func ImportUsersFromXLSXToGroup(c *gin.Context) {
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	groupID := c.Param("group_id")

	// Check permissions
	if !UserOwnsTargetGroups(user.ID, groupID) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
		respondWithError(c, http.StatusUnauthorized, "User does not have permission to create users")
		return
	}

	// Check that the group exists
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		respondWithError(c, http.StatusNotFound, ErrGroupNotFound)
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to get file: "+err.Error())
		return
	}

	// Open the file
	openedFile, err := file.Open()
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to open file: "+err.Error())
		return
	}
	defer openedFile.Close()

	// Parse the Excel file
	xlsx, err := excelize.OpenReader(openedFile)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Failed to parse XLSX file: "+err.Error())
		return
	}

	// Process all sheets
	var users []models.User
	sheetList := xlsx.GetSheetList()

	// Create temporary password (can be reset later)
	hashedPassword, err := utils.CreateDefaultPassword()
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to generate password")
		return
	}
	
	for _, sheetName := range sheetList {
		// Get all rows from the sheet
		rows, err := xlsx.GetRows(sheetName)
		if err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to read sheet: "+err.Error())
			return
		}

		if len(rows) < 2 { // At least header and one data row
			continue
		}

		// Find column indices
		var lastNameIdx, firstNameIdx, emailIdx int = -1, -1, -1
		for i, cell := range rows[0] {
			switch cell {
			case "Nom", "Nom de famille", "Last Name", "LastName":
				lastNameIdx = i
			case "Prénom 1", "Prénom", "First Name", "FirstName":
				firstNameIdx = i
			case "E-mail personnel", "E-mail", "Email", "Personal Email":
				emailIdx = i
			}
		}

		// Skip if required columns not found
		if lastNameIdx == -1 || firstNameIdx == -1 || emailIdx == -1 {
			continue
		}

		// Process data rows
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			
			// Skip empty rows
			if len(row) <= emailIdx || row[emailIdx] == "" {
				continue
			}

			// Make sure the row has enough columns
			if len(row) <= max(lastNameIdx, firstNameIdx, emailIdx) {
				continue
			}

			lastName := ""
			if lastNameIdx < len(row) {
				lastName = row[lastNameIdx]
			}
			
			firstName := ""
			if firstNameIdx < len(row) {
				firstName = row[firstNameIdx]
			}
			
			email := row[emailIdx]

			// Create user object
			newUser := models.User{
				Firstname: firstName,
				Lastname:  lastName,
				Email:     email,
				Password:  hashedPassword,
				Groups:    []*models.Group{&group},
			}

			users = append(users, newUser)
		}
	}

	if len(users) == 0 {
		respondWithError(c, http.StatusBadRequest, "No valid user data found in the file")
		return
	}

	// Create the users in the database
	for i := range users {
		if err := database.DB.Create(&users[i]).Error; err != nil {
			respondWithError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create user %s %s: %s", users[i].Firstname, users[i].Lastname, err.Error()))
			return
		}
	}

	c.JSON(http.StatusCreated, users)
}

// Helper function to find maximum value
func max(values ...int) int {
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
