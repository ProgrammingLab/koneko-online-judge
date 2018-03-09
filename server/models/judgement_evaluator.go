package models

import "github.com/gedorinku/koneko-online-judge/server/models/workers"

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
	max := 0
	maxSt := StatusAccepted
	ac := true
	for k, v := range statuses {
		if k != StatusAccepted && 0 < v {
			ac = false
			break
		}
	}
	for k, v := range statuses {
		if max < v && (!ac || k != StatusAccepted) {
			max = v
			maxSt = k
		}
	}

	return maxSt
}
