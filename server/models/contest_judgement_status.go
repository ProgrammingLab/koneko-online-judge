package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type ContestJudgementStatus struct {
	UserID    uint            `gorm:"primary_key" sql:"type:int unsigned" json:"userID"`
	ProblemID uint            `gorm:"primary_key" sql:"type:int unsigned" json:"problemID"`
	ContestID uint            `gorm:"not null" json:"contestID"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	Status    JudgementStatus `gorm:"not null; default:'0'" json:"status"`
	Point     int             `gorm:"not null; default:'0'" json:"point"`
}

func GetContestJudgementStatuses(contestID uint, userID uint) ([]ContestJudgementStatus, error) {
	res := make([]ContestJudgementStatus, 0, 0)
	db.Model(ContestJudgementStatus{}).Where("contest_id = ? AND user_id = ?", contestID, userID).Scan(&res)
	return res, nil
}

func onUpdateJudgementStatuses(contestID *uint, submission Submission) error {
	if contestID == nil {
		return nil
	}

	st := ContestJudgementStatus{}
	const query = "contest_id = ? AND user_id = ? AND problem_id = ?"
	tx := db.Begin()
	notFound := tx.Model(ContestJudgementStatus{}).Where(query, *contestID, submission.UserID, submission.ProblemID).Scan(&st).RecordNotFound()
	if notFound {
		newSt := &ContestJudgementStatus{
			ContestID: *contestID,
			UserID:    submission.UserID,
			ProblemID: submission.ProblemID,
			Status:    submission.Status,
			Point:     submission.Point,
		}
		err := tx.Create(newSt).Error
		if err != nil {
			logger.AppLog.Error(err)
			tx.Rollback()
			return err
		}

		err = tx.Model(newSt).UpdateColumns(map[string]interface{}{
			"created_at": submission.CreatedAt,
			"updated_at": submission.CreatedAt,
		}).Error
		if err != nil {
			logger.AppLog.Error(err)
			tx.Rollback()
		} else {
			tx.Commit()
		}
		return err
	}

	if st.Status == StatusAccepted || st.UpdatedAt.After(submission.UpdatedAt) {
		tx.Rollback()
		return nil
	}

	err := tx.Model(&st).UpdateColumns(map[string]interface{}{
		"updated_at": submission.CreatedAt,
		"status":     submission.Status,
		"point":      submission.Point,
	}).Error
	if err != nil {
		logger.AppLog.Error(err)
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}
