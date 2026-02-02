package cron

import (
	"database/sql"
	"fmt"
	"time"
)

// The Backend deals with storing cron job execution metadata
// for persistence.
type Backend interface {
	// Registers a job with the given name and last executed timestamp.
	// Sets attempts to 0. Does nothing if the name already exists.
	Register(name string, lastExecuted time.Time) error

	// Provides the last execution time and attempt count for the given job.
	// If a job has never been executed, it returns the timestamp for when
	// it got registered.
	LastExecutionFor(name string) (*time.Time, int, error)

	// Registers that the given job succeeded. Resets the attempt and
	// error message.
	JobSucceeded(name string) error

	// Registers a new attempt that the given job failed with the error message. Increments the attempt.
	JobFailed(name string, errmsg string) error
}

type sqlite struct {
	db *sql.DB
}

func BackendSqlite(db *sql.DB) Backend {
	return &sqlite{db: db}
}

// Register implements [Backend].
func (s *sqlite) Register(name string, lastExecuted time.Time) error {
	_, err := s.db.Exec(`
		insert or ignore into cron_jobs(name, attempt, executed_at)
		values (?, 0, ?);
	`, name, lastExecuted)
	if err != nil {
		return fmt.Errorf("sql: %w", err)
	}
	return nil
}

// LastExecutionFor implements [Backend].
func (s *sqlite) LastExecutionFor(name string) (*time.Time, int, error) {
	var (
		ts      sql.NullTime
		attempt int
	)
	row := s.db.QueryRow(`
		select executed_at, attempt
		from cron_jobs
		where name = ?;
	`, name)
	if err := row.Scan(&ts, &attempt); err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if !ts.Valid {
		return nil, attempt, nil
	}
	return ptr(ts.Time), attempt, nil
}

// JobFailed implements [Backend].
func (s *sqlite) JobFailed(name string, errmsg string) error {
	_, err := s.db.Exec(`
		insert into cron_jobs(name, attempt, last_error, executed_at)
		values (?, 1, ?, ?)
		on conflict(name) do update set
			attempt = cron_jobs.attempt + 1,
			last_error = excluded.last_error,
			executed_at = excluded.executed_at;
	`, name, errmsg, time.Now())
	if err != nil {
		return fmt.Errorf("sql: %w", err)
	}
	return nil
}

// JobSucceeded implements [Backend].
func (s *sqlite) JobSucceeded(name string) error {
	_, err := s.db.Exec(`
		insert into cron_jobs(name, attempt, last_error, executed_at)
		values (?, 0, null, ?)
		on conflict(name) do update set
			attempt = 0,
			last_error = null,
			executed_at = excluded.executed_at;
	`, name, time.Now())
	if err != nil {
		return fmt.Errorf("sql: %w", err)
	}
	return nil
}

func ptr[T any](t T) *T {
	return &t
}
