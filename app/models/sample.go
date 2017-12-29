package models

type Sample struct {
	ID          uint   `gorm:"primary_key"`
	Input       string `gorm:"type:text"`
	Output      string `gorm:"type:text"`
	Description string `gorm:"type:text"`
}
