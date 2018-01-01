package models

import (
	"archive/zip"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
	"github.com/pkg/errors"
)

const (
	mb                  = 1024 * 1024
	maxCaseFileSize     = 10
	maxCaseZipTotalSize = 100
	maxCaseFileCount    = 500
)

var (
	caseFileCountLimitExceeded     = errors.New(fmt.Sprintf("ファイルの数は%v個以下にしてください。", maxCaseFileCount))
	caseFileSizeLimitExceeded      = errors.New(fmt.Sprintf("展開後の各ファイルのサイズは%vMiB以下にしてください。", maxCaseFileSize))
	totalCaseFileSizeLimitExceeded = errors.New(fmt.Sprintf("展開後の合計ファイルサイズは%vMiB以下にしてください。", maxCaseZipTotalSize))
)

func GetBcryptCost() int {
	if revel.DevMode {
		return bcrypt.DefaultCost
	}
	//適当に調整する
	return 12
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
