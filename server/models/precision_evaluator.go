package models

import (
	"bufio"
	"math"
	"strconv"
	"strings"

	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
)

type precisionEvaluator struct {
	simple evaluator
	config *JudgementConfig
}

func newPrecisionEvaluator(config *JudgementConfig) precisionEvaluator {
	return precisionEvaluator{
		simple: newSimpleEvaluator(),
		config: config,
	}
}

func (e precisionEvaluator) next(set *CaseSet, factory func(set *CaseSet) caseSetEvaluator) caseSetEvaluator {
	if factory != nil {
		return e.simple.next(set, factory)
	}

	f := func(set *CaseSet) caseSetEvaluator {
		return newPrecisionCaseSetEvaluator(set, e.config)
	}
	return e.simple.next(set, f)
}

func (e precisionEvaluator) evaluate() (JudgementStatus, int) {
	return e.simple.evaluate()
}

func (e precisionEvaluator) remove() {
	e.simple.remove()
}

type precisionCaseSetEvaluator struct {
	setPoint int
	statuses map[JudgementStatus]int
	config   *JudgementConfig
}

func newPrecisionCaseSetEvaluator(set *CaseSet, config *JudgementConfig) *precisionCaseSetEvaluator {
	return &precisionCaseSetEvaluator{
		setPoint: set.Point,
		statuses: map[JudgementStatus]int{},
		config:   config,
	}
}

func (e *precisionCaseSetEvaluator) next(res *workers.ExecResult, testCase *TestCase) (JudgementStatus, int) {
	submission := bufio.NewScanner(strings.NewReader(res.Stdout))
	submission.Split(bufio.ScanWords)
	ans := bufio.NewScanner(strings.NewReader(testCase.Output))
	ans.Split(bufio.ScanWords)

	for submission.Scan() && ans.Scan() {
		s := submission.Text()
		a := ans.Text()
		if e.equals(s, a) {
			continue
		}

		return StatusWrongAnswer, 0
	}

	if submission.Scan() != ans.Scan() {
		return StatusWrongAnswer, 0
	}
	return StatusAccepted, 0
}

func (e *precisionCaseSetEvaluator) equals(a, b string) bool {
	if a == b {
		return true
	}

	af, err := strconv.ParseFloat(a, 64)
	if err != nil {
		return false
	}
	bf, err := strconv.ParseFloat(b, 64)
	if err != nil {
		return false
	}

	absolute := math.Abs(af - bf)
	relative := math.Abs((af - bf) / bf)
	diff := e.config.Difference
	return absolute < diff || relative < diff
}

func (e *precisionCaseSetEvaluator) evaluate() (JudgementStatus, int) {
	st := evaluateStatuses(e.statuses)
	if st == StatusAccepted {
		return st, e.setPoint
	}
	return st, 0
}
