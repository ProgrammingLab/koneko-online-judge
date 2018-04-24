package models

import (
	"reflect"
	"sort"
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

	if c := GetContestDeeply(contest.ID, nil); !deepEqualContest(*contest, *c) {
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

	{
		res, err := contest.IsWriter(334)
		if err != nil {
			t.Fatal(err)
		}
		if res {
			t.Fatalf("IsContestWriter returns true")
		}
	}

	for _, w := range contest.Writers {
		res, err := contest.IsWriter(w.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !res {
			t.Fatalf("IsContestWriter returns false")
		}
	}
}

func TestContest_Update(t *testing.T) {
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

	contest.Title = "あいうえお"
	contest.Description = "ほげほげ"
	contest.StartAt = time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local)
	contest.EndAt = time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local)
	tmp := *contest
	if err := contest.Update(); err != nil {
		t.Fatal(err)
	}
	if !deepEqualContest(tmp, *contest) {
		t.Fatalf("DeepEqual error: %+v %+v", tmp, *contest)
	}
}

func TestContest_AddParticipant(t *testing.T) {
	const writerID = 1
	contest := &Contest{
		Title:       "hogehoge",
		Description: "ぴよぴよ",
		StartAt:     time.Now(),
		EndAt:       time.Now(),
		Writers: []User{
			{ID: writerID},
		},
	}

	if err := NewContest(contest); err != nil {
		t.Fatal(err)
	}

	const participantID = 2
	if err := contest.AddParticipant(participantID); err != nil {
		t.Fatal(err)
	}

	{
		res, err := contest.IsParticipant(participantID)
		if err != nil {
			t.Fatal(err)
		}
		if !res {
			t.Errorf("IsContestParticipant(participantID) reutrns false")
		}
	}

	{
		res, err := contest.IsParticipant(writerID)
		if err != nil {
			t.Fatal(err)
		}
		if res {
			t.Errorf("IsContestParticipant(writerID) reutrns true")
		}
	}

	{
		res, err := contest.IsParticipant(334)
		if err != nil {
			t.Fatal(err)
		}
		if res {
			t.Errorf("IsContestParticipant(334) reutrns true")
		}
	}
}

func TestContest_UpdateWriters(t *testing.T) {
	contest := &Contest{
		Title:       "hogehoge",
		Description: "ぴよぴよ",
		StartAt:     time.Now(),
		EndAt:       time.Now(),
		Writers: []User{
			{ID: 1},
		},
	}

	if err := NewContest(contest); err != nil {
		t.Fatal(err)
	}

	contest.Writers = append(contest.Writers, User{ID: 2})
	if err := contest.UpdateWriters(); err != nil {
		t.Fatal(err)
	}

	res := GetContest(contest.ID)
	if contest == nil {
		t.Fatalf("GetContest(%v) returns nil", contest.ID)
	}
	res.FetchWriters()

	sort.Slice(res.Writers, func(i, j int) bool {
		return res.Writers[i].ID < res.Writers[j].ID
	})

	for i := range res.Writers {
		if contest.Writers[i].ID == res.Writers[i].ID {
			continue
		}

		t.Fatalf("mismatch writers: %+v %+v", contest.Writers[i], res.Writers[i])
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
