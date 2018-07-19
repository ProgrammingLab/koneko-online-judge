package models

import (
	"sort"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type Contest struct {
	ID                   uint                  `gorm:"primary_key" json:"id"`
	CreatedAt            time.Time             `json:"createdAt"`
	UpdatedAt            time.Time             `json:"updatedAt"`
	DeletedAt            *time.Time            `sql:"index" json:"-"`
	Title                string                `json:"title"`
	Description          string                `gorm:"type:text" json:"description"`
	StartAt              time.Time             `json:"startAt"`
	EndAt                time.Time             `json:"endAt"`
	Writers              []User                `gorm:"many2many:contests_writers;" json:"writers"`
	Participants         []User                `gorm:"many2many:contests_participants;" json:"-"`
	ContestsParticipants []ContestsParticipant `json:"participants"`
	Problems             []Problem             `json:"problems"`
	Duration             *time.Duration        `json:"duration"`
}

type ContestsParticipant struct {
	CreatedAt time.Time `gorm:"default:'1971-01-01 00:00:00'" json:"createdAt"`
	ContestID uint      `gorm:"not null" json:"contestID"`
	UserID    uint      `gorm:"not null" json:"userID"`
	User      User      `json:"user" json:"user"`
}

func (p *ContestsParticipant) FetchUser() {
	db.Model(p).Related(&p.User)
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

func GetAllContests() ([]Contest, error) {
	res := make([]Contest, 0, 0)
	err := db.Model(Contest{}).Order("id DESC").Find(&res).Error
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	return res, nil
}

func GetContest(id uint) *Contest {
	contest := &Contest{}
	notFound := db.Model(Contest{}).Where(id).Scan(contest).RecordNotFound()
	if notFound {
		return nil
	}
	return contest
}

func GetContestDeeply(id uint, session *UserSession) *Contest {
	contest := &Contest{}
	notFound := db.Model(Contest{}).Where(id).Scan(contest).RecordNotFound()
	if notFound {
		return nil
	}
	contest.FetchWriters()
	contest.FetchParticipants()
	if can, err := contest.CanViewProblems(session); err != nil {
		return nil
	} else if can {
		contest.FetchProblems()
	}
	return contest
}

func IsContestWriter(contestID, userID uint) (bool, error) {
	res := db.Limit(1).Table("contests_writers").Where("contest_id = ? AND user_id = ?", contestID, userID)
	res = res.Scan(&struct{}{})
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
	res = res.Scan(&struct{}{})
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
		"duration":    c.Duration,
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

func (c *Contest) GetStandings() ([]Score, error) {
	// TODO ユーザーごとに違うコンテスト開始時間を考慮する
	s := make([]Score, 0, 0)
	err := db.Model(c).Related(&s).Error
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	entered, err := c.getParticipantsEnteredTimeMap()
	if err != nil {
		return nil, err
	}

	sort.Slice(s, func(i, j int) bool {
		// TODO WA数の考慮
		if s[i].Point == s[j].Point {
			if c.Duration == nil {
				return s[i].UpdatedAt.Before(s[j].UpdatedAt)
			}
			pastI := s[i].UpdatedAt.Sub(entered[s[i].UserID])
			pastJ := s[j].UpdatedAt.Sub(entered[s[j].UserID])
			return pastI < pastJ
		}

		return s[i].Point > s[j].Point
	})
	for i := range s {
		score := &s[i]
		score.FetchDetails()
		sort.Slice(score.ScoreDetails, func(i, j int) bool {
			// TODO コンテスト中の問題の順番とか
			return score.ScoreDetails[i].ProblemID < score.ScoreDetails[j].ProblemID
		})
	}

	return s, nil
}

func (c *Contest) getParticipantsEnteredTimeMap() (map[uint]time.Time, error) {
	tmp := make([]ContestsParticipant, 0, 0)
	err := db.Model(ContestsParticipant{}).Where("contest_id = ?", c.ID).Scan(&tmp).Error
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	res := make(map[uint]time.Time, len(tmp))
	for _, p := range tmp {
		res[p.UserID] = p.CreatedAt
	}

	return res, nil
}

func (c *Contest) GetSubmissions(session *UserSession, limit, page int, userID, problemID *uint) ([]Submission, int, error) {
	query := make(map[string]interface{})
	isWriter, err := c.IsWriter(session.UserID)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, 0, err
	}

	ended, err := c.Ended(time.Now(), session)
	if err != nil {
		return nil, 0, err
	}
	if !isWriter && !ended {
		if userID != nil && *userID != session.UserID {
			return []Submission{}, 0, nil
		}
		userID = &session.UserID
	}
	if problemID != nil {
		problem := GetProblem(*problemID)
		if problem == nil || problem.ContestID == nil || *problem.ContestID != c.ID {
			return []Submission{}, 0, nil
		}
		query["problem_id"] = *problemID
	} else {
		query["contest_id"] = c.ID
	}
	if userID != nil {
		query["user_id"] = *userID
	}

	res := make([]Submission, 0, 0)
	err = db.Model(Submission{}).Order("id DESC").Where(query).Scan(&res).Error
	if err != nil {
		logger.AppLog.Error(err)
		return []Submission{}, 0, err
	}

	total := len(res)

	start := limit * (page - 1)
	end := limit * page
	if len(res) <= start {
		return []Submission{}, total, nil
	}

	if len(res) < end {
		end = len(res)
	}
	res = res[start:end]
	fetchSubmissionFieldsWithCache(res)

	return res, total, nil
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
	c.ContestsParticipants = make([]ContestsParticipant, 0)
	db.Model(ContestsParticipant{}).Where("contest_id = ?", c.ID).Order("user_id").Scan(&c.ContestsParticipants)
	for i := range c.ContestsParticipants {
		c.ContestsParticipants[i].FetchUser()
		c.ContestsParticipants[i].User.Email = ""
		c.Participants = append(c.Participants, c.ContestsParticipants[i].User)
	}
}

func (c *Contest) FetchProblems() {
	if c.ID == 0 || 0 < len(c.Problems) {
		return
	}

	c.Problems = make([]Problem, 0)
	db.Model(c).Related(&c.Problems, "Problems")
	for i := range c.Problems {
		c.Problems[i].FetchSamples()
		c.Problems[i].FetchCaseSets()
	}
}

func (c *Contest) Started(t time.Time, s *UserSession) (bool, error) {
	if c.Duration == nil {
		return c.StartAt.Before(t), nil
	}
	if s == nil {
		return false, nil
	}
	flg, err := c.IsParticipant(s.UserID)
	if err != nil {
		logger.AppLog.Error(err)
		return false, err
	}
	return flg, nil
}

func (c *Contest) Ended(t time.Time, s *UserSession) (bool, error) {
	if c.Duration == nil {
		return c.EndAt.Before(t), nil
	}
	if s == nil {
		return false, nil
	}

	rel := ContestsParticipant{}
	res := db.Model(ContestsParticipant{}).Where("contest_id = ? AND user_id = ?", c.ID, s.UserID).Scan(&rel)
	// コンテストに参加してない
	if res.RecordNotFound() {
		return false, nil
	}

	if err := res.Error; err != nil {
		logger.AppLog.Error(err)
		return false, err
	}

	return rel.CreatedAt.Add(*c.Duration).Before(t), nil
}

// コンテストが時刻tのとき開催中であればtrueを返します。
func (c *Contest) IsOpen(t time.Time, s *UserSession) (bool, error) {
	now := time.Now()
	started, err := c.Started(now, s)
	if err != nil {
		return false, err
	}
	ended, err := c.Ended(now, s)
	if err != nil {
		return false, err
	}

	return started && !ended, nil
}

func (c *Contest) CanEdit(s *UserSession) bool {
	if s == nil {
		return false
	}
	return CanEditContest(c.ID, s.UserID)
}

func (c *Contest) CanViewProblems(s *UserSession) (bool, error) {
	t := time.Now()
	if s == nil {
		return c.Ended(t, s)
	}

	isWriter, _ := c.IsWriter(s.UserID)
	if isWriter {
		return true, nil
	}

	isParticipant, _ := c.IsParticipant(s.UserID)
	started, err := c.Started(t, s)
	if err != nil {
		return false, err
	}
	ended, err := c.Ended(t, s)
	if err != nil {
		return false, err
	}
	return started && isParticipant || ended, nil
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
	const query = "INSERT INTO contests_participants (contest_id, user_id, created_at) VALUES (?, ?, ?)"
	if err := tx.Exec(query, c.ID, userID, time.Now()).Error; err != nil {
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
