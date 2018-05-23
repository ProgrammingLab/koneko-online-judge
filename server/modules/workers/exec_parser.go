package workers

import (
	"errors"
	"time"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type ExecResultParser struct {
	worker       *Worker
	stdoutParser OutputParser
	stderrParser OutputParser
}

var ErrExecResultParse = errors.New("exec result parse error")

func NewExecResultParser(w *Worker) ExecResultParser {
	w.stdout.Seek(0, 0)
	w.stderr.Seek(0, 0)
	return ExecResultParser{
		w,
		newReaderParser(w.stdout, w.separator),
		newReaderParser(w.stderr, w.separator),
	}
}

func (p *ExecResultParser) Next() (bool, *ExecResult, error) {
	_, stdout, err := p.stdoutParser.Next()
	if err != nil {
		logger.AppLog.Error(err)
		return false, nil, err
	}
	hasNextStdout, exitStatus, err := p.stdoutParser.Next()
	if err != nil {
		logger.AppLog.Error(err)
		return false, nil, err
	}

	// 実行ファイルのエラー出力は読み捨てる
	_, _, err = p.stderrParser.Next()
	if err != nil {
		logger.AppLog.Error(err)
		return false, nil, err
	}
	hasNextStderr, stderr, err := p.stderrParser.Next()
	if err != nil {
		logger.AppLog.Error(err)
		return false, nil, err
	}

	timeMillis, memoryUsage, err := parseTimeText(stderr)
	memoryUsage *= 1024

	status := checkStatus(timeMillis, memoryUsage, p.worker.TimeLimit, p.worker.MemoryLimit, exitStatus)

	res := &ExecResult{
		status,
		time.Duration(timeMillis) * time.Millisecond,
		memoryUsage,
		stdout,
		"",
	}
	return hasNextStdout && hasNextStderr, res, nil
}
