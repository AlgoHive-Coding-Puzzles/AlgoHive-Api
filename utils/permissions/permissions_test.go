package permissions

import (
	"api/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPermission(t *testing.T) {
	assert.True(t, HasPermission(SCOPES|CATALOGS, SCOPES))
	assert.False(t, HasPermission(SCOPES, CATALOGS))
}

func TestAddPermission(t *testing.T) {
	result := AddPermission(SCOPES, CATALOGS)
	assert.True(t, HasPermission(result, SCOPES))
	assert.True(t, HasPermission(result, CATALOGS))
}

func TestRemovePermission(t *testing.T) {
	combined := AddPermission(SCOPES, CATALOGS)
	result := RemovePermission(combined, CATALOGS)

	assert.True(t, HasPermission(result, SCOPES))
	assert.False(t, HasPermission(result, CATALOGS))
}

func TestGetAdminPermissions(t *testing.T) {
	admin := GetAdminPermissions()

	for _, perm := range []int{SCOPES, CATALOGS, GROUPS, COMPETITIONS, ROLES, OWNER} {
		assert.True(t, HasPermission(admin, perm))
	}
}

func TestRolesHavePermission(t *testing.T) {
	roles := []*models.Role{
		{Permissions: SCOPES},
		{Permissions: COMPETITIONS},
	}

	assert.True(t, RolesHavePermission(roles, SCOPES))
	assert.True(t, RolesHavePermission(roles, COMPETITIONS))
	assert.False(t, RolesHavePermission(roles, OWNER))
}

func TestMergeRolePermissions(t *testing.T) {
	roles := []*models.Role{
		{Permissions: SCOPES},
		{Permissions: CATALOGS},
	}

	merged := MergeRolePermissions(roles)

	assert.True(t, HasPermission(merged, SCOPES))
	assert.True(t, HasPermission(merged, CATALOGS))
	assert.False(t, HasPermission(merged, OWNER))
}

func TestIsStaff(t *testing.T) {
	assert.False(t, IsStaff(models.User{}))
	assert.True(t, IsStaff(models.User{Roles: []*models.Role{{Name: "Owner"}}}))
}

func TestIsOwner(t *testing.T) {
	owner := models.User{Roles: []*models.Role{{Permissions: OWNER}}}
	member := models.User{Roles: []*models.Role{{Permissions: SCOPES}}}

	assert.True(t, IsOwner(owner))
	assert.False(t, IsOwner(member))
}
