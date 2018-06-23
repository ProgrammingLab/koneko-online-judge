package models

import (
	"sort"

	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
)

type evaluator interface {
	next(set *CaseSet, factory func(set *CaseSet) caseSetEvaluator) caseSetEvaluator
	evaluate() (JudgementStatus, int)
	remove()
}

type caseSetEvaluator interface {
	next(res *workers.ExecResult, testCase *TestCase) (JudgementStatus, int)
	evaluate() (JudgementStatus, int)
}

func evaluateStatuses(statuses map[JudgementStatus]int) JudgementStatus {
	temp := make([]JudgementStatus, 0, len(statuses))
	for k, v := range statuses {
		if 0 < v {
			temp = append(temp, k)
		}
	}

	sort.Slice(temp, func(i, j int) bool {
		return temp[i] > temp[j]
	})
	return temp[0]
}

func toJudgementStatus(status workers.ExecStatus) JudgementStatus {
	switch status {
	case workers.StatusMemoryLimitExceeded:
		return StatusMemoryLimitExceeded
	case workers.StatusTimeLimitExceeded:
		return StatusTimeLimitExceeded
	case workers.StatusRuntimeError:
		return StatusRuntimeError
	case workers.StatusOutputLimitExceeded:
		return StatusOutputLimitExceeded
	default:
		return StatusUnknownError
	}
}