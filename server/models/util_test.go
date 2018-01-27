package models

import (
	"encoding/base64"
	"regexp"
	"testing"

	"reflect"

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

func TestUniqueUsers(t *testing.T) {
	inputs := [][]User{
		{
			User{ID: 1},
			User{ID: 2},
			User{ID: 3},
		},
		{
			User{ID: 1},
			User{ID: 1},
			User{ID: 2},
		},
		{
			User{ID: 1},
			User{ID: 1},
			User{ID: 2},
			User{ID: 3},
			User{ID: 3},
		},
		{
			User{ID: 3},
			User{ID: 2},
			User{ID: 3},
			User{ID: 1},
			User{ID: 1},
		},
	}
	outputs := [][]User{
		{
			User{ID: 1},
			User{ID: 2},
			User{ID: 3},
		},
		{
			User{ID: 1},
			User{ID: 2},
		},
		{
			User{ID: 1},
			User{ID: 2},
			User{ID: 3},
		},
		{
			User{ID: 3},
			User{ID: 2},
			User{ID: 1},
		},
	}

	for i, in := range inputs {
		out := UniqueUsers(in)
		if !reflect.DeepEqual(outputs[i], out) {
			t.Errorf("error on test case #%v", i)
		}
	}
}
