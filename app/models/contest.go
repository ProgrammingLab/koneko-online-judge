package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

type Contest struct {
	gorm.Model
	Title        string
	Description  string `gorm:"type:text"`
	StartAt      time.Time
	EndAt        time.Time
	Writers      []User `gorm:"many2many:contests_writers;"`
	Participants []User `gorm:"many2many:contests_participants;"`
}

func GetContest(id uint) *Contest {
	contest := &Contest{}
	notFound := db.Where(id).First(contest).RecordNotFound()
	if notFound {
		return nil
	}
	return contest
}

func GetDefaultContest(user *User) *Contest {
	contest := &Contest{
		Writers: []User{*user},
		StartAt: time.Unix(1919114514, 0),
		EndAt:   time.Unix(1919114514, 0),
	}

	return contest
}

func (c *Contest) Update() error {
	if c.ID == 0 {
		return db.Create(c).Error
	}

	return db.Save(c).Error
}

func (c *Contest) FetchWriters() {
	if c.ID == 0 || 0 < len(c.Writers) {
		return
	}

	db.Model(c).Related(&c.Writers)
}
