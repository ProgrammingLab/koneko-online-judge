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

func GetBcryptCost() int {
	if revel.DevMode {
		return bcrypt.DefaultCost
	}
	//適当に調整する
	return 12
}

// length bytesのランダムなBase64エンコードされたトークンを返す
func GenerateSecretToken(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

func checkValidZip(reader *zip.Reader) error {
	const (
		mb           = 1024 * 1024
		maxSize      = 10
		maxTotalSize = 100
		maxFileCount = 500
	)

	var total uint64

	if maxFileCount < len(reader.File) {
		return errors.New(fmt.Sprintf("ファイルの数は%v個以下にしてください。", maxFileCount))
	}

	for _, f := range reader.File {
		if f.UncompressedSize64 <= maxSize*mb {
			total += f.UncompressedSize64 / mb
			continue
		}

		return errors.New(fmt.Sprintf("展開後の各ファイルのサイズは%vMiB以下にしてください。", maxSize))
	}

	if maxTotalSize < total {
		return errors.New(fmt.Sprintf("展開後の合計ファイルサイズは%vMiB以下にしてください。", maxTotalSize))
	}

	return nil
}
