package workers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type ExecResultParser struct {
	index     int
	outputs   map[int]os.FileInfo
	outputDir string
	results   map[int]os.FileInfo
	resDir    string
	length    int
}

var ErrExecResultParse = errors.New("exec result parse error")

func NewExecResultParser(w *Worker) (ExecResultParser, error) {
	outputDir := w.HostJudgeDataDir + "/output"
	outputs, err := ioutil.ReadDir(outputDir)
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResultParser{}, err
	}
	outputMap, err := toFileInfoMap(outputs)
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResultParser{}, err
	}

	resDir := w.HostJudgeDataDir + "/status"
	results, err := ioutil.ReadDir(resDir)
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResultParser{}, err
	}
	resultMap, err := toFileInfoMap(results)
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResultParser{}, err
	}

	if len(outputMap) != len(resultMap) {
		logger.AppLog.Error(ErrExecResultParse)
		return ExecResultParser{}, ErrExecResultParse
	}

	return ExecResultParser{
		outputs:   outputMap,
		outputDir: outputDir,
		results:   resultMap,
		resDir:    resDir,
		length:    len(outputMap),
	}, nil
}

func toFileInfoMap(files []os.FileInfo) (map[int]os.FileInfo, error) {
	res := make(map[int]os.FileInfo, len(files))

	for _, f := range files {
		base := path.Base(f.Name())
		index, err := strconv.Atoi(base)
		if err != nil {
			return nil, err
		}

		res[index] = f
	}

	return res, nil
}

func (p *ExecResultParser) Next() (bool, *ExecResult, error) {
	if p.length <= p.index {
		return false, nil, nil
	}

	execRes, err := p.parseCurrentExecResult()
	if err != nil {
		p.index = p.length
		return false, nil, nil
	}

	fi, ok := p.outputs[p.index]
	if !ok {
		p.index = p.length
		return false, nil, ErrExecResultParse
	}

	f, err := os.Open(p.outputDir + "/" + fi.Name())
	if err != nil {
		logger.AppLog.Error(err)
		p.index = p.length
		return false, nil, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		logger.AppLog.Error(err)
		p.index = p.length
		return false, nil, err
	}
	execRes.Stdout = string(buf)

	p.index++

	return p.index <= p.length, &execRes, nil
}

func (p *ExecResultParser) parseCurrentExecResult() (ExecResult, error) {
	fi, ok := p.results[p.index]
	if !ok {
		logger.AppLog.Error(ErrExecResultParse)
		return ExecResult{}, ErrExecResultParse
	}

	f, err := os.Open(p.resDir + "/" + fi.Name())
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResult{}, err
	}
	defer f.Close()

	e := json.NewDecoder(f)
	res := ExecResult{}
	err = e.Decode(&res)
	if err != nil {
		logger.AppLog.Error(err)
		return ExecResult{}, err
	}

	return res, nil
}
