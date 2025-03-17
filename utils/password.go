package utils

import (
	"api/config"

	"golang.org/x/crypto/bcrypt"
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