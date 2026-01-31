package cron

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestSchedule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// inside here, the date is 2000-01-01, so it's a nice reference point

		// 23 0-20/2 * * *

		// at 04:05 on Sunday
		sched, err := NewSchedule("5 4 * * sun")
		if err != nil {
			t.Fatalf("failed to make sched: %v", err)
		}
		got := sched.NextExecution(time.Now())
		want, _ := time.Parse(time.DateTime, "2000-01-02 04:05")
		if got != want {
			t.Fatalf("mismatch: expected %s, got %s", got.Format(time.DateTime), want.Format(time.DateTime))
		}

	})
}
