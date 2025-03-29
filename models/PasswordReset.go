package models

import "time"

// PasswordReset represents a password reset request for a user
type PasswordReset struct {
    ID        string    `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
    User      User      `gorm:"foreignkey:UserID" json:"user"`
    Token     string    `gorm:"type:varchar(255);not null;unique" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}