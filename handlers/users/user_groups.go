package users

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/utils"
	"api/utils/permissions"
	"api/utils/response"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// GetUserGroups fetches the authenticated user's groups
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
        return // Error already handled by middleware
    }

    var groups []models.Group
    if err := database.DB.Model(user).Association("Groups").Find(&groups); err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to retrieve groups")
        return
    }

    c.JSON(http.StatusOK, groups)
}

// CreateUserAndAttachGroup creates a user and attaches groups to it
// @Summary Create a user and attach one or more groups
// @Description Create a new user and attach one or more groups to it
// @Tags Users
// @Accept json
// @Produce json
// @Param UserWithGroup body UserWithGroup true "User Profile with Groups"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/groups [post]
// @Security Bearer
func CreateUserAndAttachGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    // Check permissions
    if !permissions.IsStaff(user) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, ErrNoPermissionGroups)
        return
    }

    var userWithGroups UserWithGroup
    if err := c.ShouldBindJSON(&userWithGroups); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    // Validate input
    if userWithGroups.Email == "" || userWithGroups.FirstName == "" || userWithGroups.LastName == "" || len(userWithGroups.Group) == 0 {
        response.Error(c, http.StatusBadRequest, "Missing required user fields")
        return
    }

    // Use a transaction to ensure atomicity
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Check that groups exist
    var groups []models.Group
    if err := tx.Where("id IN (?)", userWithGroups.Group).Find(&groups).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }
    
    if len(groups) == 0 {
        tx.Rollback()
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

    // Create the user
    targetUser, err := createUser(userWithGroups.FirstName, userWithGroups.LastName, userWithGroups.Email)
    if err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, ErrFailedToHashPassword)
        return
    }

    if err := tx.Create(&targetUser).Error; err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to create user")
        return
    }

    // Associate groups to the user
    if err := tx.Model(targetUser).Association("Groups").Append(&groups); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to attach groups to user")
        return
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    // Fetch the complete user with associations
    var completeUser models.User
    if err := database.DB.Preload("Groups").First(&completeUser, "id = ?", targetUser.ID).Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "User created but failed to fetch complete data")
        return
    }

    c.JSON(http.StatusCreated, completeUser)
}

// CreateBulkUsersAndAttachGroup creates multiple users and attaches a group to them
// @Summary Create Bulk Users and attach a Group
// @Description Create multiple new users and attach a group to them
// @Tags Users
// @Accept json
// @Produce json
// @Param users body []models.User true "Users Profiles"
// @Param group_id path string true "Group ID"
// @Success 201 {array} models.User
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /user/group/{group_id}/bulk [post]
// @Security Bearer
func CreateBulkUsersAndAttachGroup(c *gin.Context) {
    user, err := middleware.GetUserFromRequest(c)
    if err != nil {
        return // Error already handled by middleware
    }

    groupID := c.Param("group_id")

    // Check permissions
    if !UserOwnsTargetGroups(user.ID, groupID) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, "User does not have permission to create users")
        return
    }
    
    // Check that the group exists
    var group models.Group
    if err := database.DB.First(&group, "id = ?", groupID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }
    
    // Retrieve users to be created
    var users []models.User
    if err := c.ShouldBindJSON(&users); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    if len(users) == 0 {
        response.Error(c, http.StatusBadRequest, "No users provided")
        return
    }
    
    // Use transaction for bulk operations
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    createdUsers := make([]models.User, 0, len(users))
    
    // Create the users and associate the group
    for i := range users {
        // Validate required fields
        if users[i].Email == "" || users[i].Firstname == "" || users[i].Lastname == "" {
            tx.Rollback()
            response.Error(c, http.StatusBadRequest, fmt.Sprintf("Missing required fields for user at index %d", i))
            return
        }
        
        users[i].Groups = append(users[i].Groups, &group)
        hashedPassword, err := utils.HashPassword(users[i].Password)
        if err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, ErrFailedToHashPassword)
            return
        }
        users[i].Password = hashedPassword
        users[i].HasDefaultPassword = true
        
        if err := tx.Create(&users[i]).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create user: %s", err.Error()))
            return
        }
        
        createdUsers = append(createdUsers, users[i])
    }
    
    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }
    
    c.JSON(http.StatusCreated, createdUsers)
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
        return // Error already handled by middleware
    }

    groupID := c.Param("group_id")

    // Check permissions - use correct status code for authorization
    if !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, "User does not have permission to delete users")
        return
    }

    // Check that the group exists
    var group models.Group
    if err := database.DB.First(&group, "id = ?", groupID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

    // Use transaction for multiple operations
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Get all users in the group
    var users []models.User
    if err := tx.Model(&group).Association("Users").Find(&users); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to retrieve users from group")
        return
    }

    if len(users) == 0 {
        c.JSON(http.StatusOK, gin.H{"message": "No users in group to remove"})
        return
    }
    
    // Delete all the associated users from the group first
    if err := tx.Model(&group).Association("Users").Delete(users); err != nil {
        tx.Rollback()
        response.Error(c, http.StatusInternalServerError, "Failed to delete user associations from group")
        return
    }

    // Delete all users from the users table
    for _, userToDelete := range users {
        // Clear other associations first
        if err := tx.Model(&userToDelete).Association("Roles").Clear(); err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to clear user role associations")
            return
        }
        
        if err := tx.Delete(&userToDelete).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, 
                fmt.Sprintf("Failed to delete user %s %s: %s", userToDelete.Firstname, userToDelete.Lastname, err.Error()))
            return
        }
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("Successfully removed %d users from group", len(users)),
        "count":   len(users),
    })
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
        return // Error already handled by middleware
    }

    groupID := c.Param("group_id")

    // Check permissions - use correct status code for authorization
    if !UserOwnsTargetGroups(user.ID, groupID) && !permissions.RolesHavePermission(user.Roles, permissions.OWNER) {
        response.Error(c, http.StatusForbidden, "User does not have permission to create users")
        return
    }

    // Check that the group exists
    var group models.Group
    if err := database.DB.First(&group, "id = ?", groupID).Error; err != nil {
        response.Error(c, http.StatusNotFound, ErrGroupNotFound)
        return
    }

    // Get the uploaded file
    file, err := c.FormFile("file")
    if err != nil {
        response.Error(c, http.StatusBadRequest, "Failed to get file: "+err.Error())
        return
    }

    // Open the file
    openedFile, err := file.Open()
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to open file: "+err.Error())
        return
    }
    defer openedFile.Close()

    // Parse the Excel file
    xlsx, err := excelize.OpenReader(openedFile)
    if err != nil {
        response.Error(c, http.StatusBadRequest, "Failed to parse XLSX file: "+err.Error())
        return
    }
    defer xlsx.Close()

    // Process all sheets
    var usersToCreate []models.User
    sheetList := xlsx.GetSheetList()

    // Create temporary password (can be reset later)
    hashedPassword, err := utils.CreateDefaultPassword()
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to generate password")
        return
    }
    
    for _, sheetName := range sheetList {
        // Get all rows from the sheet
        rows, err := xlsx.GetRows(sheetName)
        if err != nil {
            response.Error(c, http.StatusInternalServerError, "Failed to read sheet: "+err.Error())
            return
        }

        if len(rows) < 2 { // At least header and one data row
            continue
        }

        // Find column indices
        var lastNameIdx, firstNameIdx, emailIdx int = -1, -1, -1
        for i, cell := range rows[0] {
            switch cell {
            case "Nom", "NOM", "Nom de famille", "Last Name", "LastName":
                lastNameIdx = i
            case "Prénom 1", "Prénom", "PRENOM", "First Name", "FirstName":
                firstNameIdx = i
            case "E-mail personnel", "E-mail", "Email", "MAIL", "Personal Email":
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
                Firstname:         firstName,
                Lastname:          lastName,
                Email:             email,
                Password:          hashedPassword,
                HasDefaultPassword: true,
                Groups:            []*models.Group{&group},
            }

            usersToCreate = append(usersToCreate, newUser)
        }
    }

    if len(usersToCreate) == 0 {
        response.Error(c, http.StatusBadRequest, "No valid user data found in the file")
        return
    }

    // Use transaction for bulk operations
    tx := database.DB.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Collect duplicate email addresses
    existingEmails := make(map[string]bool)
    var duplicateEmails []string

    // Check for existing emails before creating
    for _, u := range usersToCreate {
        var count int64
        if err := tx.Model(&models.User{}).Where("email = ?", u.Email).Count(&count).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, "Failed to check for existing emails")
            return
        }
        
        if count > 0 {
            if !existingEmails[u.Email] {
                duplicateEmails = append(duplicateEmails, u.Email)
                existingEmails[u.Email] = true
            }
        }
    }

    // If duplicates were found, return an error with the list
    if len(duplicateEmails) > 0 {
        tx.Rollback()
        response.Error(c, http.StatusConflict, fmt.Sprintf("Found %d duplicate email(s): %v", 
            len(duplicateEmails), duplicateEmails))
        return
    }

    // Create the users in bulk
    createdUsers := make([]models.User, 0, len(usersToCreate))
    for i := range usersToCreate {
        if err := tx.Create(&usersToCreate[i]).Error; err != nil {
            tx.Rollback()
            response.Error(c, http.StatusInternalServerError, 
                fmt.Sprintf("Failed to create user %s %s: %s", 
                    usersToCreate[i].Firstname, usersToCreate[i].Lastname, err.Error()))
            return
        }
        createdUsers = append(createdUsers, usersToCreate[i])
    }

    if err := tx.Commit().Error; err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to commit transaction")
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": fmt.Sprintf("Successfully created %d users", len(createdUsers)),
        "users":   createdUsers,
        "count":   len(createdUsers),
    })
}

// Helper function to find maximum value
func max(values ...int) int {
    if len(values) == 0 {
        return 0
    }
    
    maxVal := values[0]
    for _, v := range values {
        if v > maxVal {
            maxVal = v
        }
    }
    return maxVal
}