package utils

import (
	"api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertRoles(t *testing.T) {
	roles := []*models.Role{
		{ID: "r1", Name: "Owner"},
		{ID: "r2", Name: "Member"},
	}

	result := ConvertRoles(roles)

	require.Len(t, result, 2)
	assert.Equal(t, "r1", result[0].ID)
	assert.Equal(t, "r2", result[1].ID)
}

func TestConvertRoles_Empty(t *testing.T) {
	result := ConvertRoles(nil)
	assert.Empty(t, result)
}

func TestConvertGroups(t *testing.T) {
	groups := []*models.Group{{ID: "g1", Name: "Group1"}}

	result := ConvertGroups(groups)

	require.Len(t, result, 1)
	assert.Equal(t, "g1", result[0].ID)
}

func TestConvertScopes(t *testing.T) {
	scopes := []*models.Scope{{ID: "s1", Name: "Scope1"}}

	result := ConvertScopes(scopes)

	require.Len(t, result, 1)
	assert.Equal(t, "s1", result[0].ID)
}

func TestContainsScope(t *testing.T) {
	scopes := []models.Scope{{ID: "s1"}, {ID: "s2"}}

	assert.True(t, ContainsScope(scopes, "s1"))
	assert.False(t, ContainsScope(scopes, "s3"))
}

func TestMarshalUnmarshalJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	data, err := MarshalJSON(payload{Name: "AlgoHive"})
	require.NoError(t, err)

	var decoded payload
	require.NoError(t, UnmarshalJSON(data, &decoded))
	assert.Equal(t, "AlgoHive", decoded.Name)
}
