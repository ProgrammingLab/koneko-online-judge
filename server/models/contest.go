package models

import (
	"sort"
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
	Problems     []Problem  `json:"problems"`
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
			return ErrUserIDIsZero
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

func GetContestDeeply(id uint, session *UserSession) *Contest {
	contest := &Contest{}
	notFound := db.Where(id).First(contest).RecordNotFound()
	if notFound {
		return nil
	}
	contest.FetchWriters()
	contest.FetchParticipants()
	if contest.CanViewProblems(session) {
		contest.FetchProblems()
	}
	return contest
}

func IsContestWriter(contestID, userID uint) (bool, error) {
	res := db.Limit(1).Table("contests_writers").Where("contest_id = ? AND user_id = ?", contestID, userID)
	res = res.First(&struct{}{})
	if res.RecordNotFound() {
		return false, nil
	}
	if res.Error != nil {
		return false, res.Error
	}

	return true, nil
}

func IsContestParticipant(contestID, userID uint) (bool, error) {
	res := db.Limit(1).Table("contests_participants").Where("contest_id = ? AND user_id = ?", contestID, userID)
	res = res.First(&struct{}{})
	if res.RecordNotFound() {
		return false, nil
	}
	if res.Error != nil {
		return false, res.Error
	}

	return true, nil
}

func CanEditContest(contestID, userID uint) bool {
	res, _ := IsContestWriter(contestID, userID)
	return res
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

func (c *Contest) UpdateWriters() error {
	if len(c.Writers) == 0 {
		return nil
	}

	cur := make([]User, 0)
	err := db.Model(c).Order("id ASC").Related(&cur, "Writers").Error
	if err != nil {
		return err
	}

	tx := db.Begin()

	for _, u := range cur {
		if err := c.removeWriterWithinTransaction(tx, u.ID); err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, u := range c.Writers {
		if err := c.addWriterWithinTransaction(tx, u.ID); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (c *Contest) GetStandings() []Score {
	s := make([]Score, 0, 0)
	db.Model(c).Related(&s)
	sort.Slice(s, func(i, j int) bool {
		if s[i].Point == s[j].Point {
			return s[i].UpdatedAt.After(s[j].UpdatedAt)
		}

		return s[i].Point < s[j].Point
	})
	for i := range s {
		score := &s[i]
		score.FetchDetails()
		sort.Slice(score.ScoreDetails, func(i, j int) bool {
			// TODO コンテスト中の問題の順番とか
			return score.ScoreDetails[i].ProblemID < score.ScoreDetails[j].ProblemID
		})
	}

	return s
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

func (c *Contest) FetchProblems() {
	if c.ID == 0 || 0 < len(c.Problems) {
		return
	}

	c.Problems = make([]Problem, 0)
	db.Model(c).Related(&c.Problems, "Problems")
}

func (c *Contest) Started() bool {
	return c.StartAt.Before(time.Now())
}

// コンテストが時刻tのとき開催中であればtrueを返します。
func (c *Contest) IsOpen(t time.Time) bool {
	return c.StartAt.Before(t) && c.EndAt.After(t)
}

func (c *Contest) CanEdit(s *UserSession) bool {
	if s == nil {
		return false
	}
	return CanEditContest(c.ID, s.UserID)
}

func (c *Contest) CanViewProblems(s *UserSession) bool {
	if s == nil {
		return false
	}

	isWriter, _ := c.IsWriter(s.UserID)
	if isWriter {
		return true
	}

	isParticipant, _ := c.IsParticipant(s.UserID)
	return c.Started() && isParticipant
}

func (c *Contest) IsWriter(userID uint) (bool, error) {
	return IsContestWriter(c.ID, userID)
}

func (c *Contest) IsParticipant(userID uint) (bool, error) {
	return IsContestParticipant(c.ID, userID)
}

func (c *Contest) AddParticipant(userID uint) error {
	return c.addParticipantTransaction(db, userID)
}

func (c *Contest) addParticipantTransaction(tx *gorm.DB, userID uint) error {
	const query = "INSERT INTO contests_participants (contest_id, user_id) VALUES (?, ?)"
	if err := tx.Exec(query, c.ID, userID).Error; err != nil {
		return err
	}

	newScore(userID, c.ID, tx)
	return tx.Error
}

func (c *Contest) addWriterWithinTransaction(tx *gorm.DB, userID uint) error {
	const query = "INSERT INTO contests_writers (contest_id, user_id) VALUES (?, ?)"
	return tx.Exec(query, c.ID, userID).Error
}

func (c *Contest) removeWriterWithinTransaction(tx *gorm.DB, userID uint) error {
	const query = "DELETE FROM contests_writers WHERE contest_id = ? AND user_id = ?"
	return tx.Exec(query, c.ID, userID).Error
}
