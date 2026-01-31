package cron

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type schedule struct {
	Minute  filterer
	Hour    filterer
	Day     filterer
	Month   filterer
	Weekday filterer
}

func (s schedule) NextExecution(ref time.Time) time.Time {
	ch := make(chan time.Time, 10)
	result := s.Minute.filter(ch)
	result = s.Hour.filter(ch)
	result = s.Day.filter(ch)
	result = s.Weekday.filter(ch)

	for i := 0; ; i++ {
		candidate := ref.Add(time.Minute * time.Duration(i))
		select {
		case ts := <-result:
			fmt.Printf("got ts %s\n", ts)
			return ts
		case ch <- candidate:
		}
	}
}

type pattern string

func (p pattern) values(start, end int) []int {
	var res []int
	if p == "*" {
		for i := start; i < end; i++ {
			res = append(res, i)
		}
	}
	if p == "20-40" {
		for i := 20; i <= 40; i++ {
			res = append(res, i)
		}
	}
	return res
}

type filterer interface {
	filter(ch <-chan time.Time) chan time.Time
}

type minute pattern

func (m minute) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 10)
	go func() {
		defer close(out)
		for t := range ch {
			for _, d := range pattern(m).values(0, 60) {
				out <- withMinute(t, d)
			}
		}
	}()
	return out
}

type hour pattern

func (h hour) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 10)
	go func() {
		defer close(out)
		for t := range ch {
			for _, d := range pattern(h).values(0, 24) {
				out <- withHour(t, d)
			}
		}
	}()
	return out
}

type day pattern

func (h day) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 10)
	go func() {
		defer close(out)
		for t := range ch {
			out <- t // TODO
			// for _, d := range pattern(h).values(0, 24) {
			// 	out <- withHour(t, d)
			// }
		}
	}()
	return out
}

type month pattern

func (h month) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 10)
	go func() {
		defer close(out)
		for t := range ch {
			out <- t // TODO
			// for _, d := range pattern(h).values(0, 24) {
			// 	out <- withHour(t, d)
			// }
		}
	}()
	return out
}

type weekday pattern

func (h weekday) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 10)
	go func() {
		defer close(out)
		for t := range ch {
			out <- t // TODO
			// for _, d := range pattern(h).values(0, 24) {
			// 	out <- withHour(t, d)
			// }
		}
	}()
	return out
}

func NewSchedule(pat string) (*schedule, error) {
	parts := strings.Split(pat, " ")
	if len(parts) != 5 {
		return nil, fmt.Errorf("wrong number of parts: expected 5, got %d", len(parts))
	}
	s := schedule{
		Minute:  minute(parts[0]),
		Hour:    hour(parts[1]),
		Day:     day(parts[2]),
		Month:   month(parts[3]),
		Weekday: weekday(parts[4]),
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

func withMinute(t time.Time, m int) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), m, t.Second(), t.Nanosecond(),
		t.Location(),
	)
}

func withHour(t time.Time, h int) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		h, t.Minute(), t.Second(), t.Nanosecond(),
		t.Location(),
	)
}

func withDay(t time.Time, d int) time.Time {
	return time.Date(
		t.Year(), t.Month(), d,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
		t.Location(),
	)
}
