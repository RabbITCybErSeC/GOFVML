// Package progress provides optional progress reporting for long-running operations.
package progress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Event represents a progress update during a long-running operation.
type Event struct {
	// Operation identifies the high-level operation (e.g., "physical acquisition").
	Operation string
	// Phase identifies the current phase (e.g., "reading", "writing", "uploading").
	Phase string
	// Current is the number of completed units (bytes, blocks, etc.).
	Current uint64
	// Total is the total number of units, or 0 if unknown.
	Total uint64
	// Target identifies what is being processed (e.g., source path, PID).
	Target string
	// Timestamp is when the event was generated.
	Timestamp time.Time
}

// Percentage returns the completion percentage, or 0 if Total is 0.
func (e Event) Percentage() float64 {
	if e.Total == 0 {
		return 0
	}
	return float64(e.Current) / float64(e.Total) * 100
}

// String returns a human-readable representation of the event.
func (e Event) String() string {
	if e.Total > 0 {
		return fmt.Sprintf("%s/%s: %s %d/%d (%.1f%%)",
			e.Operation, e.Phase, e.Target, e.Current, e.Total, e.Percentage())
	}
	return fmt.Sprintf("%s/%s: %s %d",
		e.Operation, e.Phase, e.Target, e.Current)
}

// Callback is a function that receives progress events.
type Callback func(Event)

// Reporter provides thread-safe progress reporting.
type Reporter struct {
	callback Callback
	mu       sync.Mutex
	closed   atomic.Bool
}

// NewReporter creates a new Reporter with the given callback.
// If callback is nil, events are silently dropped.
func NewReporter(callback Callback) *Reporter {
	return &Reporter{callback: callback}
}

// Report emits a progress event if the reporter is not closed.
func (r *Reporter) Report(e Event) {
	if r.closed.Load() {
		return
	}
	if r.callback == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	r.callback(e)
}

// Close marks the reporter as closed. No further events will be delivered.
func (r *Reporter) Close() {
	r.closed.Store(true)
}

// ContextReporter wraps a context and reporter to provide cancellation-aware progress.
type ContextReporter struct {
	ctx      context.Context
	reporter *Reporter
}

// NewContextReporter creates a ContextReporter.
func NewContextReporter(ctx context.Context, reporter *Reporter) *ContextReporter {
	return &ContextReporter{ctx: ctx, reporter: reporter}
}

// Report emits a progress event if the context is not cancelled.
func (cr *ContextReporter) Report(e Event) error {
	select {
	case <-cr.ctx.Done():
		return cr.ctx.Err()
	default:
	}
	cr.reporter.Report(e)
	return nil
}

// Ctx returns the underlying context.
func (cr *ContextReporter) Ctx() context.Context {
	return cr.ctx
}

// Reporter returns the underlying reporter.
func (cr *ContextReporter) Reporter() *Reporter {
	return cr.reporter
}
