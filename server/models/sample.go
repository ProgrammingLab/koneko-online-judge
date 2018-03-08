package models

type Sample struct {
	ID          uint   `gorm:"primary_key" json:"-"`
	ProblemID   uint   `gorm:"not null" json:"problemID"`
	Input       string `gorm:"type:text" json:"input"`
	Output      string `gorm:"type:text" json:"output"`
	Description string `gorm:"type:text" json:"description"`
}

func (s *Sample) Delete() {
	db.Delete(s)
}
