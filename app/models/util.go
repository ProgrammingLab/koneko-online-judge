package models

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
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
