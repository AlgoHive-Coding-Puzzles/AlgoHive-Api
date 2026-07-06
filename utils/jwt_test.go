package utils

import (
	"api/config"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withJWTConfig(t *testing.T, secret string, expirationSeconds int) {
	t.Helper()
	prevSecret, prevExp := config.JWTSecret, config.JWTExpiration
	config.JWTSecret = secret
	config.JWTExpiration = expirationSeconds
	t.Cleanup(func() {
		config.JWTSecret = prevSecret
		config.JWTExpiration = prevExp
	})
}

func TestGenerateJWT_Success(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	token, err := GenerateJWT("user-1", "user@example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Success(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	token, err := GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	claims, err := ValidateToken(token)

	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
}

// TestTokenExpiration mirrors the report's cas de test: using an expired
// token must fail validation.
func TestTokenExpiration(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	// Build a token that already expired 1 second ago.
	claims := &Claims{
		UserID: "user-1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(config.JWTSecret))
	require.NoError(t, err)

	_, err = ValidateToken(signed)

	require.Error(t, err)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	token, err := GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	// Validate against a different secret than the one used to sign.
	config.JWTSecret = "another-secret"

	_, err = ValidateToken(token)

	assert.Error(t, err)
}

// TestGenerateJWT_UniquePerCall is a regression test: two tokens generated
// for the same user within the same second used to be byte-identical
// (no jti claim), which broke rotation semantics (blacklisting the old
// token would also blacklist the freshly issued one).
func TestGenerateJWT_UniquePerCall(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	token1, err := GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)
	token2, err := GenerateJWT("user-1", "user@example.com")
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestValidateToken_Malformed(t *testing.T) {
	withJWTConfig(t, "test-secret", 3600)

	_, err := ValidateToken("not-a-jwt-token")

	assert.Error(t, err)
}
