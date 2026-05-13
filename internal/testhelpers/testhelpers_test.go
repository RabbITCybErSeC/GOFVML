package testhelpers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestCancelAfter(t *testing.T) {
	ctx, cancel := CancelAfter(t, 50*time.Millisecond)
	defer cancel()

	AssertNotCancelled(t, ctx)

	time.Sleep(100 * time.Millisecond)
	AssertCancelled(t, ctx)
}

func TestCancelImmediately(t *testing.T) {
	ctx := CancelImmediately(t)
	AssertCancelled(t, ctx)
}

func TestAssertCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	AssertCancelled(t, ctx)
}

func TestAssertNotCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	AssertNotCancelled(t, ctx)
}

func TestRunWithCancel(t *testing.T) {
	var gotCtx context.Context
	RunWithCancel(t, func(ctx context.Context, cancel context.CancelFunc) {
		gotCtx = ctx
		AssertNotCancelled(t, ctx)
	})
	// After RunWithCancel returns, the context should be cancelled.
	AssertCancelled(t, gotCtx)
}

func TestWaitForCondition(t *testing.T) {
	// Condition that becomes true after a short delay.
	var ready atomic.Bool
	go func() {
		time.Sleep(50 * time.Millisecond)
		ready.Store(true)
	}()

	if !WaitForCondition(200*time.Millisecond, ready.Load) {
		t.Error("expected condition to become true")
	}

	// Condition that never becomes true.
	if WaitForCondition(50*time.Millisecond, func() bool { return false }) {
		t.Error("expected condition to time out")
	}
}

func TestFakeSource(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	fs := NewFakeSource(data, 0x1000)

	buf := make([]byte, 5)
	n, err := fs.ReadAt(buf, 0x1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes, got %d", n)
	}
	for i, want := range data {
		if buf[i] != want {
			t.Errorf("byte[%d] = %d, want %d", i, buf[i], want)
		}
	}

	// Read beyond data should return zeros.
	buf2 := make([]byte, 3)
	n, err = fs.ReadAt(buf2, 0x1005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 bytes, got %d", n)
	}
	for i := range buf2 {
		if buf2[i] != 0 {
			t.Errorf("expected zero byte[%d], got %d", i, buf2[i])
		}
	}

	// Close should mark source as closed.
	if err := fs.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = fs.ReadAt(buf, 0x1000)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled after close, got %v", err)
	}
}
