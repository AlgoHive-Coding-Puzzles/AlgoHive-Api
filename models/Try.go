package models

// Try représente une tentative d'un utilisateur pour résoudre un puzzle dans une compétition
type Try struct {
    ID            string      `gorm:"type:uuid;default:gen_random_uuid();primary_key" json:"id"`
    PuzzleID      string      `gorm:"type:varchar(50);not null;column:puzzle_id;uniqueIndex:unique_try" json:"puzzle_id"`
    PuzzleIndex   int         `gorm:"type:integer;not null;column:puzzle_index;uniqueIndex:unique_try" json:"puzzle_index"`
    Step          int         `gorm:"type:integer;not null;uniqueIndex:unique_try" json:"step"`
    CompetitionID string      `gorm:"type:uuid;not null;column:competition_id;uniqueIndex:unique_try" json:"competition_id"`
    UserID        string      `gorm:"type:uuid;not null;column:user_id;uniqueIndex:unique_try" json:"user_id"`
    PuzzleLvl     string      `gorm:"type:varchar(50);not null;column:puzzle_lvl" json:"puzzle_lvl"`
    StartTime     string      `gorm:"type:timestamp;not null;column:start_time" json:"start_time"`
    EndTime       *string     `gorm:"type:timestamp;column:end_time" json:"end_time"`
    Attempts      int         `gorm:"type:integer;not null" json:"attempts"`
    Score         float64     `gorm:"type:numeric(15,2);not null" json:"score"`
    LastMoveTime  *string     `gorm:"type:timestamp;column:last_move_time" json:"last_move_time"`
    LastAnswer    *string     `gorm:"type:numeric(15,2);column:last_answer" json:"last_answer"`
    Competition   *Competition `gorm:"foreignKey:CompetitionID" json:"-"`
    User          *User        `gorm:"foreignKey:UserID" json:"-"`
}