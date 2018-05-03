package models

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type ContestJudgementStatus struct {
	ContestID uint            `gorm:"primary_key" sql:"type:int unsigned" json:"contestID"`
	UserID    uint            `gorm:"primary_key" sql:"type:int unsigned" json:"userID"`
	ProblemID uint            `gorm:"not null" json:"problemID"`
	CreatedAt time.Time       `json:"CreatedAt"`
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
	notFound := db.Model(ContestJudgementStatus{}).Where(query, *contestID, submission.UserID, submission.ProblemID).Scan(&st).RecordNotFound()
	if notFound {
		err := db.Create(&ContestJudgementStatus{
			ContestID: *contestID,
			UserID:    submission.UserID,
			ProblemID: submission.ProblemID,
			Status:    submission.Status,
			Point:     submission.Point,
		}).Error
		return err
	}

	if st.Status == StatusAccepted || st.UpdatedAt.After(submission.UpdatedAt) {
		return nil
	}

	err := db.Model(&st).UpdateColumns(map[string]interface{}{
		"updated_at": submission.UpdatedAt,
		"status":     submission.Status,
		"point":      submission.Point,
	}).Error
	if err != nil {
		logger.AppLog.Error(err)
	}
	return err
}
