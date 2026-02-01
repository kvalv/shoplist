package cron

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type schedule struct {
	Minute  []int
	Hour    []int
	Day     []int
	Month   []int
	Weekday []int
}

func (s schedule) NextExecution(ref time.Time) time.Time {
	ch := make(chan time.Time, 0)
	result := make(chan time.Time, 0)
	result = func(ch <-chan time.Time) chan time.Time {
		out := make(chan time.Time, 0)
		go func() {
			defer close(out)
			for t := range ch {
				out <- t // TODO
			}
		}()
		return out
	}(ch)

	result = minute(s.Minute, result)
	result = hour(s.Hour, result)
	result = day(s.Day, result)
	result = month(s.Month, result)
	result = weekday(s.Weekday, result)

	for i := 0; ; i++ {
		test := ref.Add(time.Minute * time.Duration(i))
		select {
		case ts := <-result:
			return ts
		case ch <- test:
		}
	}
}

// parses a full cron pattern, including commas, e.g. "5,10-20/2,*"
func parsePattern(p string, start, end int) ([]int, error) {
	var res []int
	for _, part := range strings.Split(p, ",") {
		values, err := parsePart(part, start, end)
		if err != nil {
			return nil, err
		}
		res = append(res, values...)
	}
	return uniq(res), nil
}

// parses a single cron pattern, e.g. "5", "0-10/2" or "*"
func parsePart(input string, lower, upper int) ([]int, error) {
	if input == "*" {
		var res []int
		for i := lower; i <= upper; i++ {
			res = append(res, i)
		}
		return res, nil
	}

	re := regexp.MustCompile(`(\d+)(-\d+)?(/\d+)?`)
	got := re.FindAllStringSubmatch(strings.TrimSpace(input), -1)
	if len(got) == 0 {
		return nil, fmt.Errorf("parseSeq: invalid pattern: %s", input)
	}

	n := mustInt(got[0][1])
	if n > upper {
		return nil, fmt.Errorf("parseSeq: value out of range")
	}
	if n < lower {
		return nil, fmt.Errorf("parseSeq: value out of range")
	}

	lower = n
	upper = lower
	step := 1

	if got[0][2] != "" {
		upper = mustInt(strings.TrimPrefix(got[0][2], "-"))
	}
	if got[0][3] != "" {
		step = mustInt(strings.TrimPrefix(got[0][3], "/"))
	}

	var res []int
	for i := lower; i <= upper; i += step {
		res = append(res, i)
	}

	return res, nil
}

func minute(accept []int, ch <-chan time.Time) chan time.Time {
	// func (m minute) filter(ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 0)
	// accept := p.values(0, 60)
	fmt.Printf("minute: accept values %v\n", accept)

	go func() {
		defer close(out)
		for t := range ch {
			if slices.Contains(accept, t.Minute()) {
				fmt.Printf("minute: accept %s\n", t)
				out <- t
			}
		}
	}()
	return out
}

func hour(accept []int, ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 0)
	fmt.Printf("hour: accept values %v\n", accept)

	go func() {
		defer close(out)
		for t := range ch {
			if slices.Contains(accept, t.Hour()) {
				fmt.Printf("hour: accept %s\n", t)
				out <- t
			}
		}
	}()
	return out
}

func day(accept []int, ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 0)
	fmt.Printf("day: accept values %v\n", accept)

	go func() {
		defer close(out)
		for t := range ch {
			if slices.Contains(accept, t.Day()) {
				fmt.Printf("day: accept %s\n", t)
				out <- t
			}
		}
	}()
	return out
}

func month(accept []int, ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 0)
	fmt.Printf("month: accept values %v\n", accept)
	go func() {
		defer close(out)
		for t := range ch {
			if slices.Contains(accept, int(t.Month())) {
				fmt.Printf("month: accept %s\n", t)
				out <- t
			}
		}
	}()
	return out
}

func weekday(accept []int, ch <-chan time.Time) chan time.Time {
	out := make(chan time.Time, 0)
	fmt.Printf("weekday: accept values %v\n", accept)
	go func() {
		defer close(out)
		for t := range ch {
			if slices.Contains(accept, int(t.Weekday())) {
				fmt.Printf("weekday: accept %s\n", t)
				out <- t
			}
		}
	}()
	return out
}

// minute | hour | day | month | weekday
// - weekday is 0-6 (sun-sat)
func NewSchedule(pat string) (*schedule, error) {
	parts := strings.Split(pat, " ")
	if len(parts) != 5 {
		return nil, fmt.Errorf("wrong number of parts: expected 5, got %d", len(parts))
	}

	var (
		s   schedule
		err error
	)

	if s.Minute, err = parsePattern(parts[0], 0, 59); err != nil {
		return nil, fmt.Errorf("invalid minute part: %w", err)
	}
	if s.Hour, err = parsePattern(parts[1], 0, 23); err != nil {
		return nil, fmt.Errorf("invalid hour part: %w", err)
	}
	if s.Day, err = parsePattern(parts[2], 1, 31); err != nil {
		return nil, fmt.Errorf("invalid day part: %w", err)
	}
	if s.Month, err = parsePattern(parts[3], 1, 12); err != nil {
		return nil, fmt.Errorf("invalid month part: %w", err)
	}
	if s.Weekday, err = parsePattern(parts[4], 0, 6); err != nil {
		return nil, fmt.Errorf("invalid weekday part: %w", err)
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

func uniq[T comparable](input []T) []T {
	var res []T
	for _, v := range input {
		if !slices.Contains(res, v) {
			res = append(res, v)
		}
	}
	return res
}

func mustInt(s string) int {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("mustInt: %v", err))
	}
	return int(n)
}
