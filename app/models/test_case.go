package models

import (
	"time"
	"archive/zip"
	"io"
)

type TestCase struct {
	ID        uint   `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CaseSetID uint   `gorm:"not null"`
	Input     string `gorm:"type:longtext; not null"`
	Output    string `gorm:"type:longtext; not null"`
}

func newTestCase(set *CaseSet, input *zip.File, output *zip.File) (*TestCase, error) {
	var (
		err     error
		in, out *string
	)

	in, err = readStringFull(input)
	if err != nil {
		return nil, err
	}

	out, err = readStringFull(output)
	if err != nil {
		return nil, err
	}

	testCase := &TestCase{
		CaseSetID: set.ID,
		Input:     *in,
		Output:    *out,
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
	db.Delete(&c)
}