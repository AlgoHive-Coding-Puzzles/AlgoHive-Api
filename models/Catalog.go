package models

type Catalog struct {
    ID          string  `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Address     string  `gorm:"type:varchar(255);not null" json:"address"`
    Name        string  `gorm:"type:varchar(100);unique;not null" json:"name"`
    Description string  `gorm:"type:varchar(255);not null" json:"description"`
    Scopes      []*Scope `gorm:"many2many:scope_catalogs;" json:"scopes"`
}