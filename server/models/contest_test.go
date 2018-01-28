package models

import (
	"reflect"
	"testing"
	"time"
)

func TestNewContest(t *testing.T) {
	contest := &Contest{
		Title:       "hogehoge",
		Description: "ぴよぴよ",
		StartAt:     time.Now(),
		EndAt:       time.Now(),
		Writers: []User{
			{ID: 1},
			{ID: 2},
		},
	}

	if err := NewContest(contest); err != nil {
		t.Fatal(err)
	}

	if c := GetContestDeeply(contest.ID); !deepEqualContest(*contest, *c) {
		t.Fatalf("DeepEqual error: GetContestDeeply %+v %+v", contest, c)
	}

	{
		c := GetContest(contest.ID)
		orig := *contest
		orig.Writers = nil
		orig.Participants = nil
		if !deepEqualContest(orig, *c) {
			t.Fatalf("DeepEqual error: GetContest %+v %+v", orig, c)
		}
	}

	if !contest.IsWriter(334) {
		t.Fatalf("IsWriter returns true")
	}

	for _, w := range contest.Writers {
		if !contest.IsWriter(w.ID) {
			t.Fatalf("IsWriter returns false")
		}
	}
}

func deepEqualContest(a, b Contest) bool {
	if !EqualTime(a.CreatedAt, b.CreatedAt) {
		return false
	}
	if !EqualTime(a.UpdatedAt, b.UpdatedAt) {
		return false
	}
	if !EqualTime(a.StartAt, b.StartAt) {
		return false
	}
	if !EqualTime(a.EndAt, b.EndAt) {
		return false
	}

	fillContestTimeZero(&a)
	fillContestTimeZero(&b)
	return reflect.DeepEqual(a, b)
}

func fillContestTimeZero(c *Contest) {
	c.CreatedAt = time.Time{}
	c.UpdatedAt = time.Time{}
	c.StartAt = time.Time{}
	c.EndAt = time.Time{}
}
