package models

type Sample struct {
	ID          uint   `gorm:"primary_key"`
	ProblemID   uint   `gorm:"not null"`
	Input       string `gorm:"type:text"`
	Output      string `gorm:"type:text"`
	Description string `gorm:"type:text"`
}

func (s *Sample) Delete() {
	db.Delete(s)
}
