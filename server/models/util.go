package models

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"time"

	"github.com/pkg/errors"
)

const (
	mb                  = 1024 * 1024
	maxCaseFileSize     = 10
	maxCaseZipTotalSize = 50
	maxCaseFileCount    = 500
)

var (
	caseFileCountLimitExceeded     = errors.New(fmt.Sprintf("ファイルの数は%v個以下にしてください。", maxCaseFileCount))
	caseFileSizeLimitExceeded      = errors.New(fmt.Sprintf("展開後の各ファイルのサイズは%vMiB以下にしてください。", maxCaseFileSize))
	totalCaseFileSizeLimitExceeded = errors.New(fmt.Sprintf("展開後の合計ファイルサイズは%vMiB以下にしてください。", maxCaseZipTotalSize))
)

func GetBcryptCost() int {
	//適当に調整する
	return 14
}

// length bytesのランダムなBase64エンコードされた文字列を返す
func GenerateRandomBase64String(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

func MaxLong(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func MaxDuration(a, b time.Duration) time.Duration {
	if a < b {
		return b
	}
	return a
}

func DefaultString(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

func checkValidZip(reader *zip.Reader) error {
	var total uint64

	if maxCaseFileCount < len(reader.File) {
		return caseFileCountLimitExceeded
	}

	for _, f := range reader.File {
		if f.UncompressedSize64 <= maxCaseFileSize*mb {
			total += f.UncompressedSize64 / mb
			continue
		}

		return caseFileSizeLimitExceeded
	}

	if maxCaseZipTotalSize < total {
		return totalCaseFileSizeLimitExceeded
	}

	return nil
}
