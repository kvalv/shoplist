package cron

import (
	"context"
	"database/sql"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestSchedule(t *testing.T) {
	cases := []struct {
		input string // minute | hour | day | month | weekday
		ref   string // yyyy-mm-dd
		want  string // yyyy-mm-dd hh:mm:ss
	}{
		{
			input: "* * * * *",
			ref:   "2000-01-01",
			want:  "2000-01-01 00:01:00",
		},
		{
			input: "20-40 * * * *",
			ref:   "2000-01-01",
			want:  "2000-01-01 00:20:00",
		},
		{
			input: "20-40 2-5 * * *",
			ref:   "2000-01-01",
			want:  "2000-01-01 02:20:00",
		},
		{
			input: "1-2 1-2 2-3 * *",
			ref:   "2000-01-01",
			want:  "2000-01-02 01:01:00",
		},
		{
			input: "5 * * * *",
			ref:   "2000-01-01",
			want:  "2000-01-01 00:05:00",
		},
		{
			// Next Friday 13th
			input: "0 0 13 * 5",
			ref:   "2000-01-01",
			want:  "2000-10-13 00:00:00",
		},
		{
			input: "23 0-20/2 * * *",
			ref:   "2000-01-01",
			want:  "2000-01-01 00:23:00", // or sth
		},
		{
			input: "@weekly",
			ref:   "1987-10-19",
			want:  "1987-10-25 00:00:00",
		},
		{
			input: "@monthly",
			ref:   "1995-07-04",
			want:  "1995-08-01 00:00:00",
		},
		{
			input: "5 4 * * mon",
			ref:   "1993-11-10",
			want:  "1993-11-15 04:05:00",
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			ref, err := time.Parse(time.DateOnly, tc.ref)
			if err != nil {
				t.Fatalf("test setup error: invalid ref date: %s", err)
			}
			want, err := time.Parse(time.DateTime, tc.want)
			if err != nil {
				t.Fatalf("test setup error: invalid want date: %s", err)
			}

			sched, err := newSchedule(tc.input)
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

func TestCron(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		if err != nil {
			t.Fatalf("failed to open db: %s", err)
		}
		defer db.Close()
		cron := New(t.Context(), BackendSqlite(db)).WithLogger(slog.New(slog.NewTextHandler(t.Output(), nil)))

		// var count int
		var count atomic.Int32
		increment := func(ctx context.Context, attempt int) error {
			count.Add(1)
			return nil
		}

		// Every hour
		if err := cron.Job("foo", "0 * * * *", increment); err != nil {
			t.Fatalf("failed to register job: %v", err)
		}
		go cron.Run()
		defer cron.Stop()

		time.Sleep(time.Hour * 3) // runs at hour 2 and hour 3

		if count.Load() != 2 {
			t.Fatalf("expected job to run 2 times, got %d", count.Load())
		}
	})
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
