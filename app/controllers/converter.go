package controllers

import (
	"time"

	"github.com/gedorinku/koneko-online-judge/app/models"
)

type badge struct {
	Style string
	Text  string
}

type Converter struct{}

var (
	badges = map[models.JudgementStatus]badge{
		models.InQueue:             {"badge-default", "WJ"},
		models.Judging:             {"badge-info", "Judging"},
		models.Accepted:            {"badge-success", "AC"},
		models.PresentationError:   {"badge-warning", "PE"},
		models.WrongAnswer:         {"badge-danger", "WA"},
		models.TimeLimitExceeded:   {"badge-warning", "TLE"},
		models.MemoryLimitExceeded: {"badge-warning", "MLE"},
		models.RuntimeError:        {"badge-warning", "RE"},
		models.CompileError:        {"badge-warning", "CE"},
		models.OutputLimitExceeded: {"badge-warning", "OLE"},
		models.UnknownError:        {"badge-primary", "UE"},
	}

	htmlDateTimeLayout = "2006-01-02T15:04:05"

	converter = &Converter{}
)

func (Converter) GetStatusBadgeStyle(status models.JudgementStatus) string {
	return badges[status].Style
}

func (Converter) GetStatusBadge(status models.JudgementStatus) string {
	return badges[status].Text
}

func (Converter) DateTime(t time.Time) string {
	return t.Format(htmlDateTimeLayout)
}
