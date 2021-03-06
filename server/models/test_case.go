package models

import (
	"archive/zip"
	"io"
	"strings"

	"github.com/jinzhu/gorm"
)

type TestCase struct {
	gorm.Model
	CaseSetID uint   `gorm:"not null"`
	Input     string `gorm:"type:longtext; not null"`
	Output    string `gorm:"type:longtext; not null"`
}

var newlineReplacer = strings.NewReplacer(
	"\r\n", "\n",
	"\r", "\n",
)

func newTestCase(set *CaseSet, input *zip.File, output *zip.File) (*TestCase, error) {
	in, err := readStringFull(input)
	if err != nil {
		return nil, err
	}

	out, err := readStringFull(output)
	if err != nil {
		return nil, err
	}

	testCase := &TestCase{
		CaseSetID: set.ID,
		Input:     newlineReplacer.Replace(*in),
		Output:    newlineReplacer.Replace(*out),
	}
	db.Create(testCase)

	return testCase, nil
}

func readStringFull(file *zip.File) (*string, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	buf := make([]byte, file.UncompressedSize64)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	result := string(buf)
	return &result, nil
}

func (c TestCase) Delete() {
	db.Model(TestCase{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"input":  "",
		"output": "",
	})
	db.Delete(TestCase{}, "id = ?", c.ID)
}

func (c TestCase) DeletePermanently() {
	db.Unscoped().Delete(TestCase{}, "id = ?", c.ID)
}
