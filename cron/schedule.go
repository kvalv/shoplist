// This file deals with parsing a cron schedule into a schedule object that is easier to work with
// when calculating the next execution time.

package cron

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type schedule struct {
	minute, hour, day, month, weekday filter
}

// filter reads a stream of time.Time objects and outputs those that match the filter.
type filter = func(t time.Time) bool

func (s schedule) NextExecution(ref time.Time) time.Time {
	for i := 1; ; i++ {
		t := ref.Add(time.Minute * time.Duration(i))

		if !every(t, s.minute, s.hour, s.day, s.month, s.weekday) {
			continue
		}
		return t
	}
}

// minute | hour | day | month | weekday
// - weekday is 0-6 (sun-sat)
func newSchedule(pat string) (*schedule, error) {
	parts := strings.Split(preprocess(pat), " ")
	if len(parts) != 5 {
		return nil, fmt.Errorf("wrong number of parts: expected 5, got %d", len(parts))
	}

	newFilter := func(accept []int, accessor func(time.Time) int) filter {
		return func(t time.Time) bool {
			return slices.Contains(accept, accessor(t))
		}
	}

	var (
		sched  schedule
		accept []int
		err    error
	)

	if accept, err = parse(parts[0], 0, 59); err != nil {
		return nil, fmt.Errorf("invalid minute part: %w", err)
	}
	sched.minute = newFilter(accept, func(t time.Time) int { return t.Minute() })

	if accept, err = parse(parts[1], 0, 23); err != nil {
		return nil, fmt.Errorf("invalid hour part: %w", err)
	}
	sched.hour = newFilter(accept, func(t time.Time) int { return t.Hour() })

	if accept, err = parse(parts[2], 1, 31); err != nil {
		return nil, fmt.Errorf("invalid day part: %w", err)
	}
	sched.day = newFilter(accept, func(t time.Time) int { return t.Day() })

	if accept, err = parse(parts[3], 1, 12); err != nil {
		return nil, fmt.Errorf("invalid month part: %w", err)
	}
	sched.month = newFilter(accept, func(t time.Time) int { return int(t.Month()) })

	if accept, err = parse(parts[4], 0, 6); err != nil {
		return nil, fmt.Errorf("invalid weekday part: %w", err)
	}
	sched.weekday = newFilter(accept, func(t time.Time) int { return int(t.Weekday()) })

	return &sched, nil
}

// normalizes some special patterns into the traditional cron format
func preprocess(pat string) string {
	pat = strings.ToLower(strings.TrimSpace(pat))
	switch pat {
	case "@hourly":
		return "0 * * * *"
	case "@daily":
		return "0 0 * * *"
	case "@weekly":
		return "0 0 * * 0"
	case "@monthly":
		return "0 0 1 * *"
	}

	for ord, name := range []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"} {
		if ok, _ := regexp.MatchString(name+"$", pat); ok {
			return fmt.Sprintf("%s%d", strings.TrimSuffix(pat, name), ord)
		}
	}

	return pat
}

// parses a full cron pattern, including commas, e.g. "5,10-20/2,*"
func parse(p string, start, end int) ([]int, error) {
	var res []int
	for part := range strings.SplitSeq(p, ",") {
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
func every(t time.Time, filters ...filter) bool {
	for _, f := range filters {
		if !f(t) {
			return false
		}
	}
	return true
}
