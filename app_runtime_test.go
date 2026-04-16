package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCleanupLoopStopsAfterCancel(t *testing.T) {
	app := &App{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	var calls atomic.Int32

	go func() {
		app.runCleanupLoop(ctx, 5*time.Millisecond, "test cleanup", func(context.Context) error {
			calls.Add(1)
			return nil
		})
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("cleanup loop did not stop after cancel")
	}

	if calls.Load() == 0 {
		t.Fatal("expected cleanup loop to run at least once before cancellation")
	}
}
