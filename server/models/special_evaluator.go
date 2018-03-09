package models

import (
	"strconv"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/models/workers"
)

type specialEvaluator struct {
	verifier   *workers.Worker
	simple     *simpleEvaluator
	config     *JudgementConfig
	submission *Submission
}

func newSpecialEvaluator(config *JudgementConfig, submission *Submission) (specialEvaluator, error) {
	e := specialEvaluator{
		verifier:   nil,
		simple:     newSimpleEvaluator(),
		config:     config,
		submission: submission,
	}
	return e, nil
}

func (e specialEvaluator) next(set *CaseSet, factory func(set *CaseSet) caseSetEvaluator) caseSetEvaluator {
	if factory != nil {
		return e.simple.next(set, factory)
	}

	f := func(set *CaseSet) caseSetEvaluator {
		return newSpecialCaseSetEvaluator(e.verifier, e.config, e.submission)
	}
	return e.simple.next(set, f)
}

func (e specialEvaluator) remove() {
	if e.verifier != nil {
		e.verifier.Remove()
	}
}

type specialCaseSetEvaluator struct {
	point      int
	statuses   map[JudgementStatus]int
	verifier   *workers.Worker
	config     *JudgementConfig
	submission *Submission
}

func newSpecialCaseSetEvaluator(verifier *workers.Worker, config *JudgementConfig, submission *Submission) *specialCaseSetEvaluator {
	return &specialCaseSetEvaluator{
		statuses:   map[JudgementStatus]int{},
		verifier:   verifier,
		config:     config,
		submission: submission,
	}
}

func (e *specialCaseSetEvaluator) next(res *workers.ExecResult, testCase *TestCase) (JudgementStatus, int) {
	compiled, compileRes := compile(*e.config.JudgeSourceCode, e.config.Language)
	if compiled == nil || compileRes == nil {
		return UnknownError, 0
	}
	defer compiled.Remove()
	if compileRes.Status != workers.StatusFinished {
		return CompileError, 0
	}

	l := e.submission.Language
	const (
		input      = "in"
		output     = "out"
		userOutput = "submission"
	)
	cmd := append(e.config.Language.GetExecCommandSlice(), input, output, userOutput, l.FileName)
	w, err := workers.NewWorker(imageNamePrefix+e.config.Language.ImageName, compileTimeLimit, compileMemoryLimit, cmd)
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return UnknownError, 0
	}
	defer w.Remove()

	w.CopyContentToContainer([]byte(testCase.Input), input)
	w.CopyContentToContainer([]byte(testCase.Output), output)
	w.CopyContentToContainer([]byte(res.Stdout), userOutput)
	w.CopyContentToContainer([]byte(e.submission.SourceCode), l.FileName)

	judged, err := w.Run("")
	if err != nil {
		logger.AppLog.Errorf("error: %+v", err)
		return UnknownError, 0
	}

	point, _ := strconv.Atoi(judged.Stdout)
	if judged.Status == workers.StatusFinished {
		return Accepted, point
	}
	return WrongAnswer, 0
}

func (e *specialCaseSetEvaluator) evaluate() (JudgementStatus, int) {
	st := evaluateStatuses(e.statuses)
	if st == Accepted {
		return st, e.point
	}
	return st, 0
}
