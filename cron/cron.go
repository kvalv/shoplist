package cron

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

// Creates a new Cron instance.
// It does not handle long-running tasks very well.
// Still, should be good enough for now
func New(ctx context.Context, backend Backend) *Cron {
	ctx, cancel := context.WithCancel(ctx)
	c := &Cron{
		ctx:     ctx,
		cancel:  cancel,
		backend: backend,
		log:     log.New(io.Discard, "", 0),
	}

	return c
}
func (c *Cron) WithLogger(logger *log.Logger) *Cron {
	c.log = logger
	return c
}

func (c *Cron) Run() {
	c.logf("started")
	defer c.logf("done")
	c.wg.Add(1)
	defer c.wg.Done()
	tick := time.Tick(time.Second)
	done := c.ctx.Done()
	for {
		select {
		case <-done:
			return
		case <-tick:
			c.poll()
		}
	}
}

func (c *Cron) Stop() {
	c.logf("Stop() called")
	c.cancel()
	c.wg.Wait()
	c.logf("Stop() done")
}

type Cron struct {
	ctx          context.Context
	cancel       context.CancelFunc
	backend      Backend
	jobs         []job
	log          *log.Logger
	wg           sync.WaitGroup
	pollInterval time.Duration
}

func (c *Cron) Job(
	name string,
	pattern string,
	handler handler,
) error {
	if name == "" {
		return fmt.Errorf("job name cannot be empty")
	}
	sched, err := newSchedule(pattern)
	if err != nil {
		return fmt.Errorf("failed to parse schedule pattern: %w", err)
	}
	if err := c.backend.Register(name, time.Now()); err != nil {
		return fmt.Errorf("failed to register job: %w", err)
	}
	c.jobs = append(c.jobs, job{
		name:     name,
		schedule: sched,
		handler:  handler,
	})
	return nil
}

// Registers a new job. If the registration fails due to invalid pattern or name, it panics.
func (c *Cron) Must(
	name string,
	pattern string,
	handler handler,
) {
	if err := c.Job(name, pattern, handler); err != nil {
		panic("cron: failed to register job: " + err.Error())
	}
}

type job struct {
	name string
	*schedule
	handler
}

type handler func(ctx context.Context, attempt int) error

func (c *Cron) poll() {
	ctx := c.ctx
	if ctx.Err() != nil {
		return
	}

	now := time.Now()
	var wg sync.WaitGroup
	defer wg.Wait()
	c.wg.Add(1)
	defer c.wg.Done()
	for _, job := range c.jobs {
		last, attempt, err := c.backend.LastExecutionFor(job.name)
		if err != nil {
			c.logf("failed to fetch last executed: %s", err)
			continue
		}
		if last == nil {
			panic("Did not expect last to be nil when there's no error.")
		}

		next := job.schedule.NextExecution(*last)
		// Allow exact match; happens during test, probably never in reality.
		if !next.Before(now) {
			continue
		}

		wg.Go(func() {
			if ctx.Err() != nil {
				c.logf("job %q: context done, skipping execution", job.name)
				return
			}
			c.logf("executing job %q", job.name)
			t0 := time.Now()
			if err := job.handler(ctx, attempt+1); err != nil {
				c.logf("job %q failed. Registering failed attempt. Error: %s.", job.name, err)
				if err := c.backend.JobFailed(job.name, err.Error()); err != nil {
					c.logf("failed to register failed attempt: %s", err)
				}
				return
			}
			if err := c.backend.JobSucceeded(job.name); err != nil {
				c.logf("failed to register successful attempt: %s", err)
			}
			c.logf("job %q done, took %s, next execution %s",
				job.name,
				time.Since(t0),
				job.schedule.NextExecution(time.Now()).Format(time.DateTime),
			)
			{
				lastRan, _, err := c.backend.LastExecutionFor(job.name)
				if err != nil {
					c.logf("job %q: failed to fetch last executed after success: %s", job.name, err)
					return
				}
				tsNext := job.schedule.NextExecution(*lastRan)
				c.logf("job %q: next execution scheduled at %s", job.name, tsNext.Format(time.DateTime))
			}
		})
	}
}

func (c *Cron) logf(format string, a ...any) {
	c.log.Printf(format, a...)
}
