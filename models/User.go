package models

import (
	"time"
)

// User represents a user in the system with authentication details
type User struct {
    ID                 string     `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Firstname          string     `gorm:"type:varchar(50);not null" json:"first_name"`
    Lastname           string     `gorm:"type:varchar(50);not null" json:"last_name"`
    Email              string     `gorm:"type:varchar(255);unique;not null" json:"email"`
    Password           string     `gorm:"type:varchar(255);not null" json:"password"`
    LastConnected      *time.Time `gorm:"type:timestamp" json:"last_connected"` 
    Blocked            bool       `gorm:"not null;default:false" json:"blocked"`
    HasDefaultPassword bool       `json:"has_default_password" gorm:"column:has_default_password;default:true"`
    Groups             []*Group   `gorm:"many2many:user_groups;" json:"groups"`
    Roles              []*Role    `gorm:"many2many:user_roles;" json:"roles"`
}