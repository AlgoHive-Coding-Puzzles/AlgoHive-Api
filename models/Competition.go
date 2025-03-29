package models

// Competition represents a competition with puzzles that can be attempted by users
type Competition struct {
    ID           string  `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    Title        string  `gorm:"type:varchar(100);not null" json:"title"`
    Description  string  `gorm:"type:varchar(255)" json:"description"`
    CatalogID    string  `gorm:"type:uuid;not null;column:catalog_id" json:"catalog_id"`
    CatalogTheme string  `gorm:"type:varchar(50);not null;column:catalog_theme" json:"catalog_theme"`
    Show         bool    `gorm:"not null;default:false" json:"show"`
    Finished     bool    `gorm:"not null;default:false" json:"finished"`
    Catalog      *Catalog `gorm:"foreignKey:CatalogID" json:"catalog"`
    Groups       []*Group `gorm:"many2many:competition_groups;" json:"groups"`
    Tries        []*Try   `gorm:"foreignKey:CompetitionID" json:"tries"`
}