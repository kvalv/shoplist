package cron

import (
	"database/sql"
	"testing"
	"testing/synctest"
	"time"

	_ "modernc.org/sqlite"
)

func TestBackendSqlite(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open db: %s", err)
	}
	backend := BackendSqlite(db)

	synctest.Test(t, func(t *testing.T) {
		if err := backend.JobSucceeded("job"); err != nil {
			t.Fatalf("failed to register successful attempt: %s", err)
		}

		time.Sleep(time.Hour * 2)

		ts, attempt, err := backend.LastExecutionFor("job")
		if err != nil {
			t.Fatalf("failed to get last execution: %s", err)
		}
		if ts == nil {
			t.Fatalf("expected timestamp to be non-nil")
		}

		want := "2000-01-01 00:00:00"
		if got := ts.Format(time.DateTime); got != want {
			t.Fatalf("unexpected timestamp: got %q, want %q", got, want)
		}
		if attempt != 0 {
			t.Fatalf("unexpected attempt: got %d, want 0", attempt)
		}

		diff := time.Since(*ts)
		if diff != time.Hour*2 {
			t.Fatalf("timestamp difference too small: got %s, want %s", diff, time.Hour*2)
		}

		// register a failed attempt
		if err := backend.JobFailed("job", "some error"); err != nil {
			t.Fatalf("failed to register failed attempt: %s", err)
		}

		time.Sleep(time.Hour * 3)
		ts, attempt, err = backend.LastExecutionFor("job")
		if err != nil {
			t.Fatalf("failed to get last execution: %s", err)
		}
		if ts == nil {
			t.Fatalf("expected timestamp to be non-nil")
		}
		if attempt != 1 {
			t.Fatalf("unexpected attempt: got %d, want 1", attempt)
		}
		diff = time.Since(*ts)
		if diff != time.Hour*3 {
			t.Fatalf("timestamp difference too small: got %s, want %s", diff, time.Hour*3)
		}
	})
}
