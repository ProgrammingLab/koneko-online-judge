package models

import (
	"encoding/base64"
	"regexp"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateRandomBase64String(t *testing.T) {
	const length = 32
	regex := regexp.MustCompile(`^[A-Za-z0-9/+]*=*$`)
	res := GenerateRandomBase64String(length)
	if len(res) != base64.StdEncoding.EncodedLen(length) {
		t.Errorf("incorrect length: %v", res)
	}
	if !regex.MatchString(res) {
		t.Errorf("no match base64: %v", res)
	}
}

func TestGetBcryptCost(t *testing.T) {
	c := GetBcryptCost()
	if c < bcrypt.MinCost || bcrypt.MaxCost < c {
		t.Errorf("cost is out of renge: %v", c)
	}
}
