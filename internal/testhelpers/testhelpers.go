// Package testhelpers provides utilities for testing long-running operations.
package testhelpers

import (
	"context"
	"sync"
	"testing"
	"time"
)

// CancelAfter returns a context that is cancelled after the given duration.
// The caller is responsible for calling the cancel function.
func CancelAfter(t *testing.T, d time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	return ctx, cancel
}

// CancelImmediately returns a context that is already cancelled.
func CancelImmediately(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// AssertCancelled asserts that the given context is cancelled.
func AssertCancelled(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Error("expected context to be cancelled")
	}
}

// AssertNotCancelled asserts that the given context is not cancelled.
func AssertNotCancelled(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Error("expected context to not be cancelled")
	default:
		// expected
	}
}

// RunWithCancel runs the given function with a cancellable context and
// cancels it after the function returns. Useful for ensuring cleanup.
func RunWithCancel(t *testing.T, fn func(ctx context.Context, cancel context.CancelFunc)) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fn(ctx, cancel)
}

// WaitForCondition waits up to timeout for the condition to return true.
// It returns true if the condition was met, false if the timeout expired.
func WaitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// FakeSource provides an in-memory source for testing acquisition without
// privileged devices.
type FakeSource struct {
	mu     sync.RWMutex
	data   map[uint64]byte
	closed bool
}

// NewFakeSource creates a FakeSource with the given data starting at offset.
func NewFakeSource(data []byte, startOffset uint64) *FakeSource {
	fs := &FakeSource{data: make(map[uint64]byte)}
	for i, b := range data {
		fs.data[startOffset+uint64(i)] = b
	}
	return fs
}

// ReadAt reads bytes from the fake source at the given offset.
func (fs *FakeSource) ReadAt(p []byte, off uint64) (int, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.closed {
		return 0, context.Canceled
	}

	n := 0
	for i := range p {
		if b, ok := fs.data[off+uint64(i)]; ok {
			p[i] = b
			n++
		} else {
			// Return zeros for unreadable addresses.
			p[i] = 0
			n++
		}
	}
	return n, nil
}

// Close marks the fake source as closed.
func (fs *FakeSource) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.closed = true
	return nil
}
