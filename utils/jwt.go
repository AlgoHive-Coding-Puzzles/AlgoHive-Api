package utils

import (
	"api/config"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims
type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

// newTokenID generates a random JWT ID (jti), so two tokens issued for the
// same user within the same second (e.g. during a refresh) are never
// identical and can be individually blacklisted.
func newTokenID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Extremely unlikely; fall back to a timestamp-derived value rather
        // than failing token generation entirely.
        return time.Now().Format(time.RFC3339Nano)
    }
    return hex.EncodeToString(b)
}

// GenerateJWT generates a JWT token for a given user
func GenerateJWT(userID, email string) (string, error) {
    expirationTime := time.Now().Add(time.Duration(config.JWTExpiration) * time.Second)

    claims := &Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        newTokenID(),
            ExpiresAt: jwt.NewNumericDate(expirationTime),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    
    tokenString, err := token.SignedString([]byte(config.JWTSecret))
    if err != nil {
        return "", err
    }
    
    return tokenString, nil
}

// ValidateToken validates the JWT token
func ValidateToken(tokenString string) (*Claims, error) {
    claims := &Claims{}

    tokenString = strings.TrimSpace(tokenString)
    
    token, err := jwt.ParseWithClaims(
        tokenString,
        claims,
        func(token *jwt.Token) (interface{}, error) {
            return []byte(config.JWTSecret), nil
        },
    )
    
    if err != nil {
        return nil, err
    }
    
    if !token.Valid {
        return nil, errors.New("invalid token")
    }
    
    return claims, nil
}