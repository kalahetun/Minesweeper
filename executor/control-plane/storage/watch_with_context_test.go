package storage

import (
	"context"
	"testing"
	"time"
)

// TestWatchWithContext ensures that WatchWithContext returns a channel that
// is closed when the provided context is canceled.
func TestWatchWithContext(t *testing.T) {
	s := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	ch := s.WatchWithContext(ctx)

	// Ensure the channel is not closed immediately
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed prematurely")
		}
	default:
	}

	// Cancel the context and expect the channel to be closed shortly
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel not closed after context cancellation")
		}
		// closed as expected
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for channel to close after context cancellation")
	}
}
