package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_And_CheckPasswordHash(t *testing.T) {
	hash, err := HashPassword("SuperSecret1")
	require.NoError(t, err)
	assert.NotEqual(t, "SuperSecret1", hash)

	assert.True(t, CheckPasswordHash("SuperSecret1", hash))
}

func TestCheckPasswordHash_WrongPassword(t *testing.T) {
	hash, err := HashPassword("SuperSecret1")
	require.NoError(t, err)

	assert.False(t, CheckPasswordHash("WrongPassword1", hash))
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid password", "Abcdefg1", false},
		{"too short", "Ab1defg", true},
		{"too long", "A1" + string(make([]byte, 100)), true},
		{"missing uppercase", "abcdefg1", true},
		{"missing lowercase", "ABCDEFG1", true},
		{"missing number", "Abcdefgh", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
