package cron

import (
	"testing"
	"time"
)

func TestSchedule(t *testing.T) {
	// synctest.Test(t, func(t *testing.T) {
	// inside here, the date is 2000-01-01, so it's a nice reference point

	ref, _ := time.Parse(time.DateOnly, "2000-01-01")

	cases := []struct {
		input string
		want  string // yyyy-mm-dd hh:mm:ss
	}{
		{
			input: "* * * * *",
			want:  "2000-01-01 00:00:00",
		},
		// {
		// 	input: "5 4 * * sun",
		// 	want:  "2000-01-02 04:05",
		// },
	}

	for _, tc := range cases {
		want, err := time.Parse(time.DateTime, tc.want)
		if err != nil {
			t.Fatalf("test setup error: invalid date: %s", err)
		}

		// at 04:05 on Sunday
		sched, err := NewSchedule(tc.input)
		if err != nil {
			t.Fatalf("failed to make sched: %v", err)
		}
		t.Logf("ref is %s", ref)
		got := sched.NextExecution(ref)
		t.Logf("got is %s -- right bfore tunrt", got)
		if got != want {
			t.Fatalf("mismatch: expected %s, got %s", got.Format(time.DateTime), want.Format(time.DateTime))
		}
	}
	// })
}
