package utils

import (
	"api/config"
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
    MinPasswordLength = 8
    MaxPasswordLength = 100
)

// HashPassword hashes a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CreateDefaultPassword creates a default password
func CreateDefaultPassword() (string, error) {
	return HashPassword(config.DefaultPassword)
}

// CheckPasswordHash checks if a password is correct
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword checks if the password meets the requirements
// - At least 8 characters
// - At least 1 uppercase letter
// - At least 1 lowercase letter
// - At least 1 number
// - Maximum 100 characters
func ValidatePassword(password string) error {
    if len(password) < MinPasswordLength {
        return errors.New("password must be at least 8 characters long")
    }

    if len(password) > MaxPasswordLength {
        return errors.New("password must be less than 100 characters long")
    }

    var (
        hasUpper   bool
        hasLower   bool
        hasNumber  bool
    )

    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsNumber(char):
            hasNumber = true
        }
    }

    if !hasUpper {
        return errors.New("password must contain at least one uppercase letter")
    }
    if !hasLower {
        return errors.New("password must contain at least one lowercase letter")
    }
    if !hasNumber {
        return errors.New("password must contain at least one number")
    }

    return nil
}