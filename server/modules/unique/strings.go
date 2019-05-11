package unique

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/big"

	"github.com/ProgrammingLab/koneko-online-judge/server/logger"
)

// length bytesのランダムなBase64エンコードされた文字列を返す
func GenerateRandomBase64String(length int) string {
	bytes := make([]byte, length)
	_, err := crand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

// 長さがlengthのランダムな文字列を返す
func GenerateRandomBase62String(length int) (string, error) {
	const (
		alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	)
	maxRand := int64(len(alphaNumeric))
	chars := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := crand.Int(crand.Reader, big.NewInt(maxRand))
		if err != nil {
			logger.AppLog.Errorf("error: %+v", err)
			return "", err
		}
		chars[i] = alphaNumeric[n.Int64()]
	}
	return string(chars), nil
}
