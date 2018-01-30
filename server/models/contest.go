package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type Contest struct {
	ID           uint       `gorm:"primary_key" json:"id"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `sql:"index" json:"-"`
	Title        string     `json:"title"`
	Description  string     `gorm:"type:text" json:"description"`
	StartAt      time.Time  `json:"startAt"`
	EndAt        time.Time  `json:"endAt"`
	Writers      []User     `gorm:"many2many:contests_writers;" json:"writers"`
	Participants []User     `gorm:"many2many:contests_participants;" json:"participants"`
}

func NewContest(out *Contest) error {
	writers := out.Writers
	out.Writers = nil
	out.Participants = nil
	tx := db.Begin()
	if err := tx.Create(out).Error; err != nil {
		tx.Rollback()
		return err
	}

	for i, w := range writers {
		if w.ID == 0 {
			tx.Rollback()
			return UserIDIsZeroError
		}
		if err := out.addWriterWithinTransaction(tx, w.ID); err != nil {
			tx.Rollback()
			return err
		}

		u := GetUser(w.ID, false)
		if u == nil {
			err := tx.Error
			tx.Rollback()
			if err == nil {
				logger.AppLog.Error("unknown error")
				return errors.New("something wrong")
			}
			logger.AppLog.Error(err)
			return err
		}
		writers[i] = *u
	}
	tx.Commit()
	out.Writers = writers
	out.Participants = make([]User, 0)
	return nil
}

func GetContest(id uint) *Contest {
	contest := &Contest{}
	notFound := db.Where(id).First(contest).RecordNotFound()
	if notFound {
		return nil
	}
	return contest
}

func GetContestDeeply(id uint) *Contest {
	contest := &Contest{}
	notFound := db.Where(id).First(contest).RecordNotFound()
	if notFound {
		return nil
	}
	contest.FetchWriters()
	contest.FetchParticipants()
	return contest
}

func (c *Contest) Update() error {
	if c.ID == 0 {
		return db.Create(c).Error
	}

	query := map[string]interface{}{
		"title":       c.Title,
		"description": c.Description,
		"startAt":     c.StartAt,
		"endAt":       c.EndAt,
	}
	return db.Model(&Contest{ID: c.ID}).Updates(query).Error
}

func (c *Contest) FetchWriters() {
	if c.ID == 0 || 0 < len(c.Writers) {
		return
	}

	c.Writers = make([]User, 0)
	db.Model(c).Related(&c.Writers, "Writers")
	for i := range c.Writers {
		c.Writers[i].Email = ""
	}
}

func (c *Contest) FetchParticipants() {
	if c.ID == 0 || 0 < len(c.Participants) {
		return
	}

	c.Participants = make([]User, 0)
	db.Model(c).Related(&c.Participants, "Participants")
	for i := range c.Participants {
		c.Participants[i].Email = ""
	}
}

func (c *Contest) IsWriter(userID uint) bool {
	const query = "SELECT TOP (1) * FROM contests_writers WHERE contest_id = ? AND user_id = ?"
	notFound := db.Raw(query, c.ID, userID).RecordNotFound()
	return !notFound
}

func (c *Contest) IsParticipant(userID uint) (bool, error) {
	res := db.Limit(1).Table("contests_participants").Where("contest_id = ? AND user_id = ?", c.ID, userID)
	res = res.First(&struct{}{})
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return false, res.Error
	}

	return !res.RecordNotFound(), nil
}

func (c *Contest) AddParticipant(userID uint) error {
	return c.addParticipantTransaction(db, userID)
}

func (c *Contest) addParticipantTransaction(tx *gorm.DB, userID uint) error {
	const query = "INSERT INTO contests_participants (contest_id, user_id) VALUES (?, ?)"
	return tx.Exec(query, c.ID, userID).Error
}

func (c *Contest) addWriterWithinTransaction(tx *gorm.DB, userID uint) error {
	const query = "INSERT INTO contests_writers (contest_id, user_id) VALUES (?, ?)"
	return tx.Exec(query, c.ID, userID).Error
}
