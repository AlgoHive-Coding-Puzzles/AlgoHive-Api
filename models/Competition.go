package models

type Competition struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
	Title           string    `gorm:"type:varchar(100);not null;unique" json:"title"`
	Description     string    `gorm:"type:text;not null" json:"description"`
	Finished        bool      `gorm:"not null" json:"finished"`
	Show            bool      `gorm:"not null" json:"show"`
	CatalogTheme        string    `gorm:"type:varchar(50);not null;column:catalog_theme" json:"catalog_theme"`
	CatalogID string    `gorm:"type:uuid;not null;column:catalog_id" json:"catalog_id"`
	Catalog  *Catalog   `gorm:"foreignKey:CatalogID" json:"catalog,omitempty"`
	Groups         []*Group   `gorm:"many2many:competition_groups;" json:"groups,omitempty"`
	Tries          []*Try     `gorm:"foreignKey:CompetitionID" json:"tries,omitempty"`
}