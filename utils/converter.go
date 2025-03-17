package utils

import (
	"api/models"
	"bytes"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

// convertRoles converts []*models.Role to []models.Role
func ConvertRoles(roles []*models.Role) []models.Role {
    var result []models.Role
    for _, role := range roles {
        result = append(result, *role)
    }
    return result
}

// convertGroups converts []*models.Group to []models.Group
func ConvertGroups(groups []*models.Group) []models.Group {
    var result []models.Group
    for _, group := range groups {
        result = append(result, *group)
    }
    return result
}

// convertScopes converts []*models.Scope to []models.Scope
func ConvertScopes(scopes []*models.Scope) []models.Scope {
	var result []models.Scope
	for _, scope := range scopes {
		result = append(result, *scope)
	}
	return result
}

func ContainsScope(scopes []models.Scope, scopeId string) bool {
	for _, s := range scopes {
		if s.ID == scopeId {
			return true
		}
	}
	return false
}

func DisplayBodyContent(c *gin.Context) {
	// Print the body
	bodyData, _ := io.ReadAll(c.Request.Body)
	log.Printf("Incoming request body: %s", string(bodyData))
	// We need to restore the body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyData))
}