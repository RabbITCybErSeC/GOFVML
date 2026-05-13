package progress

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventPercentage(t *testing.T) {
	tests := []struct {
		name    string
		current uint64
		total   uint64
		want    float64
	}{
		{"zero_total", 50, 0, 0},
		{"half", 50, 100, 50},
		{"full", 100, 100, 100},
		{"quarter", 25, 100, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Current: tt.current, Total: tt.total}
			if got := e.Percentage(); got != tt.want {
				t.Errorf("Percentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventString(t *testing.T) {
	e := Event{
		Operation: "physical acquisition",
		Phase:     "reading",
		Target:    "/dev/mem",
		Current:   50,
		Total:     100,
	}
	got := e.String()
	want := "physical acquisition/reading: /dev/mem 50/100 (50.0%)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	e2 := Event{
		Operation: "upload",
		Phase:     "transferring",
		Target:    "http://example.com/dump.lime",
		Current:   1024,
		Total:     0,
	}
	got2 := e2.String()
	want2 := "upload/transferring: http://example.com/dump.lime 1024"
	if got2 != want2 {
		t.Errorf("String() = %q, want %q", got2, want2)
	}
}

func TestReporter(t *testing.T) {
	var events []Event
	var mu sync.Mutex
	cb := func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}

	r := NewReporter(cb)
	r.Report(Event{Operation: "test", Current: 1})
	r.Report(Event{Operation: "test", Current: 2})

	mu.Lock()
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	mu.Unlock()

	r.Close()
	r.Report(Event{Operation: "test", Current: 3}) // should be dropped

	mu.Lock()
	if len(events) != 2 {
		t.Errorf("expected 2 events after close, got %d", len(events))
	}
	mu.Unlock()
}

func TestReporterNilCallback(t *testing.T) {
	r := NewReporter(nil)
	// Should not panic.
	r.Report(Event{Operation: "test", Current: 1})
	r.Close()
}

func TestContextReporter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []Event
	var mu sync.Mutex
	cb := func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	}

	r := NewReporter(cb)
	cr := NewContextReporter(ctx, r)

	err := cr.Report(Event{Operation: "test", Current: 1})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	cancel()

	// Allow cancellation to propagate.
	time.Sleep(10 * time.Millisecond)

	err = cr.Report(Event{Operation: "test", Current: 2})
	if err == nil {
		t.Error("expected error after cancellation")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestEventTimestamp(t *testing.T) {
	var got Event
	cb := func(e Event) {
		got = e
	}

	r := NewReporter(cb)
	r.Report(Event{Operation: "test", Current: 1})

	if got.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}
