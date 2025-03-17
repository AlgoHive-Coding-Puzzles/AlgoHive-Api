package models

type Input struct {
    ID          string    `gorm:"type:uuid;default:gen_random_uuid();primary_key"`
    PuzzleID    string    `gorm:"type:varchar(50);not null"`
    Content     string    `gorm:"type:text;not null"`
    SolutionOne string    `gorm:"type:varchar(255);not null"`
    SolutionTwo string    `gorm:"type:varchar(255);not null"`
    UserID      string    `gorm:"type:uuid;not null"`
}