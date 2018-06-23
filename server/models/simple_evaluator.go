package models

import (
	"strings"

	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
)

type simpleEvaluator struct {
	point    int
	statuses map[JudgementStatus]int
	lastSet  caseSetEvaluator
}

func newSimpleEvaluator() *simpleEvaluator {
	return &simpleEvaluator{
		statuses: map[JudgementStatus]int{},
	}
}

func (e *simpleEvaluator) next(set *CaseSet, factory func(set *CaseSet) caseSetEvaluator) caseSetEvaluator {
	if e.lastSet != nil {
		st, pt := e.lastSet.evaluate()
		e.point += pt
		e.statuses[st]++
	}

	if set == nil {
		return nil
	}

	if factory == nil {
		e.lastSet = newSimpleCaseSetEvaluator(set)
	} else {
		e.lastSet = factory(set)
	}
	return e.lastSet
}

func (e *simpleEvaluator) evaluate() (JudgementStatus, int) {
	if e.lastSet == nil {
		return StatusUnknownError, 0
	}

	e.next(nil, nil)
	st := evaluateStatuses(e.statuses)
	return st, e.point
}

func (e *simpleEvaluator) remove() {}

type simpleCaseSetEvaluator struct {
	setPoint int
	statuses map[JudgementStatus]int
}

func newSimpleCaseSetEvaluator(set *CaseSet) *simpleCaseSetEvaluator {
	return &simpleCaseSetEvaluator{
		setPoint: set.Point,
		statuses: map[JudgementStatus]int{},
	}
}

func (e *simpleCaseSetEvaluator) next(res *workers.ExecResult, testCase *TestCase) (JudgementStatus, int) {
	// スペシャルジャッジではないので、ケースごとの点数は無視される
	st := func() JudgementStatus {
		if res == nil {
			return StatusUnknownError
		}

		if res.Status != workers.StatusFinished {
			return toJudgementStatus(res.Status)
		}

		if res.Stdout == testCase.Output {
			return StatusAccepted
		}
		if strings.TrimSpace(res.Stdout) == strings.TrimSpace(testCase.Output) {
			return StatusPresentationError
		}
		return StatusWrongAnswer
	}()

	e.statuses[st]++
	return st, 0
}

func (e *simpleCaseSetEvaluator) evaluate() (JudgementStatus, int) {
	st := evaluateStatuses(e.statuses)
	if st == StatusAccepted {
		return st, e.setPoint
	}
	return st, 0
}
