package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	t.Run("cancelling works", func(t *testing.T) {
		counter := 0
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		scheduler := NewTask(ctx, &wg, "dummy", 10*time.Millisecond, func() {
			counter++
		})
		scheduler.Run()
		time.Sleep(5 * time.Millisecond)
		cancel()
		wg.Wait()
		if counter != 0 {
			t.Errorf("expected counter to be 0 after cancellation, got %d", counter)
		}
	})
	t.Run("task executes correctly", func(t *testing.T) {
		counter := 0
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		scheduler := NewTask(ctx, nil, "running", 10*time.Millisecond, func() {
			counter++
		})

		scheduler.Run()
		time.Sleep(35 * time.Millisecond)

		if counter < 3 {
			t.Errorf("expected counter to be at least 3, got %d", counter)
		}
	})
}
