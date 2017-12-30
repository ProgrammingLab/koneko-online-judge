package models

import (
	"archive/zip"
	"bytes"
	"strconv"
	"strings"
	"time"
	"regexp"
	"github.com/pkg/errors"
)

type CaseSet struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	ProblemID uint `gorm:"not null"`
	Point     int  `gorm:"not null; default:'0'"`
	TestCases []TestCase
}

const (
	inputFilePrefix  = "input"
	outputFilePrefix = "output"
)

var (
	inputFileRegex  = regexp.MustCompile(inputFilePrefix + `(\d+)-(\d+)\.txt`)
	outputFileRegex = regexp.MustCompile(outputFilePrefix + `(\d+)-(\d+)\.txt`)
)

func newCaseSets(problem *Problem, archive []byte) ([]*CaseSet, error) {
	if problem == nil {
		return nil, errors.New("problemがnilです。")
	}
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	inputs, outputs := getCaseFilese(r)
	if inputs == nil || outputs == nil {
		return nil, errors.New("ファイルの命名かディレクトリの構造が正しくありません。")
	}

	inputSets := checkCaseFileNaming(inputs, inputFilePrefix)
	outputSets := checkCaseFileNaming(outputs, outputFilePrefix)
	if inputSets == nil || outputSets == nil {
		return nil, errors.New("ファイルの命名が正しくありません。")
	}

	c := len(inputSets)
	if c != len(outputSets) {
		return nil, errors.New("ファイルの命名が正しくありません。")
	}

	result := make([]*CaseSet, c)
	for i := 1; i <= c; i++ {
		result[i-1], err = newCaseSet(problem, inputSets[i], outputSets[i])
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func newCaseSet(problem *Problem, inputs map[int]*zip.File, outputs map[int]*zip.File) (*CaseSet, error) {
	c := len(inputs)
	if c != len(outputs) {
		return nil, errors.New("ファイルの命名が正しくありません。")
	}

	caseSet := &CaseSet{ProblemID: problem.ID}
	db.Create(caseSet)

	result := make([]*TestCase, c)
	for i := 1; i <= c; i++ {
		var err error
		result[i-1], err = newTestCase(caseSet, inputs[i], outputs[i])

		if err != nil {
			caseSet.Delete()
			return nil, err
		}
	}

	return caseSet, nil
}

func (s *CaseSet) Delete() {
	s.FetchTestCases()
	for _, c := range s.TestCases {
		c.Delete()
	}
}

func (s *CaseSet) FetchTestCases() {
	db.Model(s).Related(&s.TestCases)
}

func checkCaseFileNaming(files []*zip.File, prefix string) map[int]map[int]*zip.File {
	result := make(map[int]map[int]*zip.File)
	for _, f := range files {
		i, j := parseCaseFileName(f.Name, prefix)
		if i <= 0 || j <= 0 {
			return nil
		}
		if _, ok := result[i]; !ok {
			result[i] = make(map[int]*zip.File)
		}
		result[i][j] = f
	}

	y := len(result)
	for i := 1; i <= y; i++ {
		if _, ok := result[i]; !ok {
			return nil
		}

		x := len(result[i])
		for j := 1; j <= x; j++ {
			if _, ok := result[i][j]; !ok {
				return nil
			}
		}
	}

	return result
}

func getCaseFilese(reader *zip.Reader) ([]*zip.File, []*zip.File) {
	inputs := make([]*zip.File, 0, len(reader.File)/2)
	outputs := make([]*zip.File, 0, len(reader.File)/2)
	for _, f := range reader.File {
		switch {
		case f.FileInfo().IsDir():
			return nil, nil
		case inputFileRegex.MatchString(f.Name):
			inputs = append(inputs, f)
		case outputFileRegex.MatchString(f.Name):
			outputs = append(outputs, f)
		}
	}

	return inputs, outputs
}

// ケースファイルの名前をパースする。入力は`prefix(\d+)-(\d+)\.txt`を満たすこと!
func parseCaseFileName(name string, prefix string) (int, int) {
	hyphen := strings.IndexByte(name, '-')
	dot := strings.IndexByte(name, '.')
	i, _ := strconv.Atoi(name[len(prefix):hyphen])
	j, _ := strconv.Atoi(name[hyphen+1:dot])
	return i, j
}
