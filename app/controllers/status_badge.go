package controllers

import "github.com/gedorinku/koneko-online-judge/app/models"

type badge struct {
	Style string
	Text  string
}

type BadgeConverter struct{}

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

	Converter = &BadgeConverter{}
)

func (BadgeConverter) GetStatusBadgeStyle(status models.JudgementStatus) string {
	return badges[status].Style
}

func (BadgeConverter) GetStatusBadge(status models.JudgementStatus) string {
	return badges[status].Text
}
