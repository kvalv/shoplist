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
		input string // minute | hour | day | month | weekday
		want  string // yyyy-mm-dd hh:mm:ss
	}{
		{
			input: "* * * * *",
			want:  "2000-01-01 00:00:00",
		},
		{
			input: "20-40 * * * *",
			want:  "2000-01-01 00:20:00",
		},
		{
			input: "20-40 2-5 * * *",
			want:  "2000-01-01 02:20:00",
		},
		{
			input: "1-2 1-2 2-3 * *",
			want:  "2000-01-02 01:01:00",
		},
		{
			input: "5 * * * *",
			want:  "2000-01-01 00:05:00",
		},
		{
			// Next Friday 13th
			input: "0 0 13 * 5",
			want:  "2000-10-13 00:00:00",
		},
		{
			input: "23 0-20/2 * * *",
			want:  "2000-01-01 00:23:00", // or sth
		},
		// {
		// 	input: "5 4 * * sun",
		// 	want:  "2000-01-02 04:05",
		// },
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			want, err := time.Parse(time.DateTime, tc.want)
			if err != nil {
				t.Fatalf("xtest setup error: invalid date: %s", err)
			}
			defer func() {
				t.Logf("want is %s", want.Format(time.DateTime))
			}()

			// at 04:05 on Sunday
			sched, err := NewSchedule(tc.input)
			if err != nil {
				t.Fatalf("failed to make sched: %v", err)
			}
			got := sched.NextExecution(ref)
			if got != want {
				t.Fatalf(
					"mismatch: \n\twant %s, \n\tgot  %s",
					want.Format(time.DateTime),
					got.Format(time.DateTime),
				)
			}
		})
	}
}

func TestStep(t *testing.T) {
	cases := []struct {
		input string
		want  []int
	}{
		{"10", []int{10}},
		{"2-9/2", []int{2, 4, 6, 8}},
		{"*", []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parsePart(tc.input, 0, 12)
			if err != nil {
				t.Fatalf("parsePart failed: %v", err)
			}
			expectSlicesEq(t, tc.want, got)
		})
	}
}

func expectSlicesEq(t *testing.T, want, got []int) {
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("mismatch at index %d: want %d, got %d", i, want[i], got[i])
		}
	}
}
