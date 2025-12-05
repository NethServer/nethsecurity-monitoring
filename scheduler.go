package main

import (
	"context"
	"log/slog"
	"time"
)

type Task struct {
	ctx      context.Context
	name     string
	duration time.Duration
	handler  func()
}

type Scheduler interface {
	Run()
}

func NewTask(ctx context.Context, name string, duration time.Duration, handler func()) Scheduler {
	return Task{
		ctx:      ctx,
		name:     name,
		duration: duration,
		handler:  handler,
	}
}

func (t Task) Run() {
	tick := time.NewTicker(t.duration)
	slog.Info("Scheduler started", "task", t.name)

	go func() {
		for {
			select {
			case <-tick.C:
				slog.Debug("Executing scheduled task", "task", t.name)
				t.handler()
			case <-t.ctx.Done():
				tick.Stop()
				slog.Info("Scheduler stopped", "task", t.name)
				return
			}
		}
	}()
}
