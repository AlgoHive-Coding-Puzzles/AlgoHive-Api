package models

type Role struct {
    ID          string    `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Name        string    `gorm:"type:varchar(50);unique;not null" json:"name"`
    Permissions int       `gorm:"not null;default:0" json:"permissions"`
    Users       []*User   `gorm:"many2many:user_roles;" json:"users"`
    Scopes      []*Scope  `gorm:"many2many:role_scopes;" json:"scopes"`
}