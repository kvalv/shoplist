package cron

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type schedule struct {
	Minute  pattern
	Hour    pattern
	Day     pattern
	Month   pattern
	Weekday pattern
}

func (s schedule) NextExecution(ref time.Time) time.Time {
	want, _ := time.Parse(time.DateTime, "2000-01-02 04:05")
	return want

}

type pattern string

func NewSchedule(pat string) (*schedule, error) {
	parts := strings.Split(pat, " ")
	if len(parts) != 5 {
		return nil, fmt.Errorf("wrong number of parts: expected 5, got %d", len(parts))
	}
	s := schedule{
		Minute:  pattern(parts[0]),
		Hour:    pattern(parts[1]),
		Day:     pattern(parts[2]),
		Month:   pattern(parts[3]),
		Weekday: pattern(parts[4]),
	}

	return &s, nil
}

type cron struct {
	db *sql.DB
}

func New(db *sql.DB) *cron {
	return &cron{
		db: db,
	}
}

type Callback func(ctx context.Context, attempt int) error

type Job struct {
}

func (j Job) NextExecution() time.Time {
	return time.Now()
}

func (c *cron) Register(
	name string,
	pattern string,
	f Callback,
) Job {
	panic("todo")
}
