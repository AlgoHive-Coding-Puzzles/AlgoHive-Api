package models

type Group struct {
    ID           string        `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Name         string        `gorm:"type:varchar(50);not null" json:"name"`
    Description  string        `gorm:"type:varchar(255)" json:"description"`
    ScopeID      string        `gorm:"type:uuid;not null" json:"scope_id"`
    Users        []*User       `gorm:"many2many:user_groups;" json:"users"`
    Competitions []*Competition `gorm:"many2many:competition_groups;" json:"competitions"`
}