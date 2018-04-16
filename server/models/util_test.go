package models

import (
	"encoding/base64"
	"reflect"
	"regexp"
	"testing"
	"time"

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

func TestEqualTime(t *testing.T) {
	now := time.Now()
	inputs := []struct{ A, B time.Time }{
		{now, now},
		{
			time.Date(2018, 1, 1, 12, 0, 0, 0, time.Local),
			time.Date(2018, 1, 1, 12, 0, 0, int(100*time.Millisecond), time.Local),
		},
		{
			time.Date(2018, 1, 1, 12, 0, 0, 0, time.Local),
			time.Date(2018, 1, 1, 11, 59, 59, int(800*time.Millisecond), time.Local),
		},
		{
			time.Date(2018, 1, 1, 12, 0, 0, 0, time.Local),
			time.Date(2018, 1, 1, 11, 59, 59, int(100*time.Millisecond), time.Local),
		},
		{
			time.Date(2018, 1, 1, 12, 0, 0, int(400*time.Millisecond), time.Local),
			time.Date(2018, 1, 1, 11, 59, 59, int(600*time.Millisecond), time.Local),
		},
	}
	outputs := []bool{
		true,
		true,
		true,
		false,
		true,
	}

	for i, in := range inputs {
		if EqualTime(in.A, in.B) != outputs[i] {
			t.Errorf("error on test case #%v", i)
		}
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
