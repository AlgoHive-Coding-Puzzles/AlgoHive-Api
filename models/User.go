package models

import (
	"time"
)

type User struct {
    ID            string     `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Firstname     string     `gorm:"type:varchar(50);not null" json:"firstname"`
    Lastname      string     `gorm:"type:varchar(50);not null" json:"lastname"`
    Email         string     `gorm:"type:varchar(255);unique;not null" json:"email"`
    Password      string     `gorm:"type:varchar(255);not null" json:"password"`
    LastConnected *time.Time `gorm:"type:timestamp" json:"last_connected"` 
    Blocked       bool       `gorm:"not null;default:false" json:"blocked"`
    Groups        []*Group   `gorm:"many2many:user_groups;" json:"groups"`
    Roles         []*Role    `gorm:"many2many:user_roles;" json:"roles"`
}