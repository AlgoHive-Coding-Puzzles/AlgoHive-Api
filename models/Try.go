package models

// Try represents a user's attempt at solving a puzzle in a competition
type Try struct {
    ID            string        `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    UserID        string        `gorm:"type:uuid;not null;column:user_id" json:"user_id"`
    CompetitionID string        `gorm:"type:uuid;not null;column:competition_id" json:"competition_id"`
    PuzzleID      string        `gorm:"type:varchar(255);not null;column:puzzle_id" json:"puzzle_id"`
    PuzzleIndex   int           `gorm:"type:integer;not null;column:puzzle_index" json:"puzzle_index"`
    PuzzleLvl     string        `gorm:"type:varchar(255);not null;column:puzzle_lvl" json:"puzzle_lvl"`
    Step          int           `gorm:"type:integer;not null" json:"step"`
    StartTime     string      `gorm:"type:timestamp;not null;column:start_time" json:"start_time"`
    EndTime       *string     `gorm:"type:timestamp;column:end_time" json:"end_time"`
    Attempts      int         `gorm:"type:integer;not null" json:"attempts"`
    Score         float64     `gorm:"type:numeric(15,2);not null" json:"score"`
    LastMoveTime  *string     `gorm:"type:timestamp;column:last_move_time" json:"last_move_time"`
    LastAnswer    *string     `gorm:"type:numeric(15,2);column:last_answer" json:"last_answer"`
    Competition   *Competition `gorm:"foreignKey:CompetitionID" json:"-"`
    User          *User        `gorm:"foreignKey:UserID" json:"user"`
}