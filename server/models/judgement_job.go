package models

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
	"github.com/gedorinku/koneko-online-judge/server/modules/workers"
	"github.com/gocraft/work"
	"github.com/pkg/errors"
)

type judgementJob struct {
	submissionID uint
	submission   *Submission

	compiled *workers.Worker
}

const (
	imageNamePrefix    = "koneko-online-judge-image-"
	compileTimeLimit   = 20 * time.Second
	compileMemoryLimit = 512 * 1024 * 1024
)

var ErrParseOutput = errors.New("stdout parse error")

func judge(submissionID uint) error {
	_, err := enqueuer.Enqueue(judgementJobName, work.Q{submissionJobArgKey: submissionID})
	if err != nil {
		logger.AppLog.Errorf("job error: %+v", err)
	}
	return err
}

func compile(sourceCode string, language *Language) (*workers.Worker, *workers.ExecResult) {
	cmd := language.GetCompileCommandSlice()
	w, err := workers.NewTimeoutWorker(imageNamePrefix+language.ImageName, compileTimeLimit, compileMemoryLimit, cmd)
	if err != nil {
		logger.AppLog.Errorf("compile: container create error %+v", err)
		return nil, nil
	}

	err = w.CopyContentToContainer([]byte(sourceCode), workers.Workspace+language.FileName)
	if err != nil {
		logger.AppLog.Errorf("compile: docker cp %+v", err)
		return nil, nil
	}

	res, err := w.Run("", true)
	if err != nil {
		logger.AppLog.Errorf("compile: container attach error %+v", err)
		return nil, nil
	}

	return w, res
}

func (j *judgementJob) Run() {
	defer func() {
		if err := recover(); err != nil {
			logger.AppLog.Errorf("%+v", err)
		}
	}()

	defer j.Close()
	j.submission = GetSubmission(j.submissionID)
	if j.submission == nil {
		logger.AppLog.Infof("submission(id = %v) is deleted", j.submissionID)
		return
	}
	j.submission.SetStatus(StatusJudging)
	j.submission.FetchLanguage()
	j.submission.FetchProblem()
	j.submission.Problem.FetchJudgementConfig()
	j.submission.FetchJudgeSetResults(false)
	var (
		execTime    time.Duration
		memoryUsage int64
		point       = 0
		finalStatus = StatusUnknownError
	)

	defer func() {
		query := map[string]interface{}{
			"point":        point,
			"status":       finalStatus,
			"exec_time":    execTime,
			"memory_usage": memoryUsage,
		}
		db.Model(&Submission{ID: j.submission.ID}).Updates(query)
		onUpdateJudgementStatuses(j.submission.Problem.ContestID, *j.submission)
	}()

	var eval evaluator
	switch j.submission.Problem.JudgeType {
	case JudgeTypeNormal:
		eval = newSimpleEvaluator()
	case JudgeTypePrecision:
		eval = newPrecisionEvaluator(j.submission.Problem.JudgementConfig)
	case JudgeTypeSpecial:
		var err error
		eval, err = newSpecialEvaluator(j.submission.Problem.JudgementConfig, j.submission)
		if err != nil {
			logger.AppLog.Errorf("judge source code compile error: %+v", err)
			finalStatus = StatusUnknownError
			markAs(j.submission.JudgeSetResults, finalStatus)
			return
		}
	default:
		logger.AppLog.Errorf("%v is not implemented", j.submission.Problem.JudgeType)
		finalStatus = StatusUnknownError
		markAs(j.submission.JudgeSetResults, finalStatus)
		return
	}

	defer eval.remove()

	var compileRes *workers.ExecResult
	j.compiled, compileRes = compile(j.submission.SourceCode[:], &j.submission.Language)
	if j.compiled == nil || compileRes == nil {
		finalStatus = StatusUnknownError
		markAs(j.submission.JudgeSetResults, finalStatus)
	} else {
		logger.AppLog.Debugf("%v %v", compileRes.Status, compileRes.Stderr)

		if compileRes.Status != workers.StatusFinished {
			finalStatus = StatusCompileError
			markAs(j.submission.JudgeSetResults, finalStatus)
			logger.AppLog.Debugf("compile error: worker status %v", compileRes.Status, compileRes.Stderr)
		} else {
			for _, r := range j.submission.JudgeSetResults {
				r.FetchCaseSet()
				setEval := eval.next(&r.CaseSet, nil)
				w, err := j.executeCaseSet(setEval, &r)
				if err != nil {
					logger.AppLog.Error(err)
					break
				}
				j.judgeCaseSet(w, setEval, &r)
				execTime = MaxDuration(execTime, r.ExecTime)
				memoryUsage = MaxLong(memoryUsage, r.MemoryUsage)
			}
		}
	}

	if finalStatus != StatusCompileError {
		finalStatus, point = eval.evaluate()
	}

	j.submission.Point = point
	j.submission.Status = finalStatus
	j.submission.ExecTime = execTime
	j.submission.MemoryUsage = memoryUsage

	if j.submission.Problem.ContestID != nil {
		p := &j.submission.Problem
		p.FetchContest()
		writer, err := p.Contest.IsWriter(j.submission.UserID)
		if err != nil {
			logger.AppLog.Errorf("error %+v", err)
			return
		}
		open, err := p.Contest.IsOpen(j.submission.CreatedAt, &UserSession{UserID: j.submission.UserID})
		if err != nil {
			logger.AppLog.Error(err)
			return
		}
		if open && !writer {
			updateScore(j.submission, *j.submission.Problem.ContestID)
		}
	}
}

func (j *judgementJob) Close() {
	if j.compiled == nil {
		return
	}
	j.compiled.Remove()
}

func markAs(setResults []JudgeSetResult, status JudgementStatus) {
	for _, s := range setResults {
		s.FetchJudgeResults(false)
		db.Model(s).Update("status", status)
		for _, r := range s.JudgeResults {
			db.Model(r).Update("status", status)
		}
	}
}

func (j *judgementJob) executeCaseSet(evaluator caseSetEvaluator, result *JudgeSetResult) (*workers.Worker, error) {
	result.FetchJudgeResults(false)

	w, err := j.createJudgementWorker(result.JudgeResults)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}

	res, err := w.Run("", false)
	if err != nil {
		logger.AppLog.Error(err)
		return nil, err
	}
	logger.AppLog.Debug(res)

	return w, err
}

func (j *judgementJob) judgeCaseSet(w *workers.Worker, evaluator caseSetEvaluator, setResult *JudgeSetResult) {
	defer w.Remove()

	var (
		maxExecTime    time.Duration
		maxMemoryUsage int64
		hasErr         bool
	)

	p, err := workers.NewExecResultParser(w)
	if err != nil {
		logger.AppLog.Error(err)
		hasErr = true
	}
	results := setResult.JudgeResults

	for i, r := range results {
		r.FetchTestCase()
		var (
			has bool
			res *workers.ExecResult
			err error
		)
		if !hasErr {
			has, res, err = p.Next()
			logger.AppLog.Debug(i)
			if err != nil {
				logger.AppLog.Error(err)
				res = nil
				hasErr = true
			} else if !has && i != len(results)-1 {
				logger.AppLog.Error(ErrParseOutput)
				res = nil
				hasErr = true
			}
		}

		r.Status, _ = evaluator.next(res, &r.TestCase)
		if err != nil || res == nil {
			r.Status = StatusUnknownError
		}
		if res != nil {
			r.ExecTime = res.ExecTime
			r.MemoryUsage = res.MemoryUsage / 1024
		}

		query := map[string]interface{}{
			"status":       r.Status,
			"exec_time":    r.ExecTime,
			"memory_usage": r.MemoryUsage,
		}
		db.Model(&JudgeResult{ID: r.ID}).Updates(query)

		maxExecTime = MaxDuration(maxExecTime, r.ExecTime)
		maxMemoryUsage = MaxLong(maxMemoryUsage, r.MemoryUsage)
	}

	setResult.Status, setResult.Point = evaluator.evaluate()
	setResult.ExecTime = maxExecTime
	setResult.MemoryUsage = maxMemoryUsage

	query := map[string]interface{}{
		"point":        setResult.Point,
		"status":       setResult.Status,
		"exec_time":    setResult.ExecTime,
		"memory_usage": setResult.MemoryUsage,
	}
	db.Model(&JudgeSetResult{ID: setResult.ID}).Updates(query)
}

func (j *judgementJob) createJudgementWorker(results []JudgeResult) (*workers.Worker, error) {
	problem := &j.submission.Problem
	language := &j.submission.Language
	cmd := language.GetExecCommandSlice()
	w, err := workers.NewJudgementWorker(imageNamePrefix+language.ImageName, problem.TimeLimit, int64(problem.MemoryLimit*1024*1024), cmd)
	if err != nil {
		logger.AppLog.Errorf("exec: container create error %+v", err)
		w.Remove()
		return nil, err
	}

	inputDir := w.HostJudgeDataDir + "/input"
	err = os.Mkdir(inputDir, 0700)
	if err != nil {
		logger.AppLog.Error(err)
		w.Remove()
		return nil, err
	}

	shuffleJudgeResults(results)

	for i, r := range results {
		r.FetchTestCase()
		name := inputDir + "/" + strconv.Itoa(i)
		err := func() error {
			f, err := os.Create(name)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.WriteString(r.TestCase.Input)

			return err
		}()
		if err != nil {
			logger.AppLog.Error(err)
			w.Remove()
			return nil, err
		}
	}

	err = j.compiled.CopyTo(workers.Workspace+language.ExeFileName, w)
	if err != nil {
		logger.AppLog.Errorf("exec: docker cp error %+v", err)
		w.Remove()
		return nil, err
	}

	return w, nil
}

func shuffleJudgeResults(results []JudgeResult) error {
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		logger.AppLog.Error(err)
		return err
	}

	r := rand.New(rand.NewSource(seed.Int64()))

	n := len(results)
	for i := n - 1; i >= 0; i-- {
		j := r.Intn(i + 1)
		results[i], results[j] = results[j], results[i]
	}

	return nil
}
