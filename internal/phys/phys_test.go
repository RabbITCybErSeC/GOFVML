package phys

import "testing"

func TestRangeLen(t *testing.T) {
	r := Range{Start: 0x1000, End: 0x2000}
	if got, want := r.Len(), uint64(0x1000); got != want {
		t.Errorf("Len() = %x, want %x", got, want)
	}
}

func TestRangeIsEmpty(t *testing.T) {
	empty := Range{Start: 0x1000, End: 0x1000}
	if !empty.IsEmpty() {
		t.Error("expected empty range")
	}
	nonEmpty := Range{Start: 0x1000, End: 0x2000}
	if nonEmpty.IsEmpty() {
		t.Error("expected non-empty range")
	}
}

func TestRangeContains(t *testing.T) {
	r := Range{Start: 0x1000, End: 0x2000}
	if !r.Contains(0x1000) {
		t.Error("expected 0x1000 to be contained")
	}
	if r.Contains(0x2000) {
		t.Error("expected 0x2000 to not be contained (exclusive end)")
	}
}

func TestRangeOverlaps(t *testing.T) {
	a := Range{Start: 0x1000, End: 0x2000}
	b := Range{Start: 0x1800, End: 0x2800}
	if !a.Overlaps(b) {
		t.Error("expected overlap")
	}
	c := Range{Start: 0x2000, End: 0x3000}
	if a.Overlaps(c) {
		t.Error("expected no overlap")
	}
}

func TestRangeAdjacent(t *testing.T) {
	a := Range{Start: 0x1000, End: 0x2000}
	b := Range{Start: 0x2000, End: 0x3000}
	if !a.Adjacent(b) {
		t.Error("expected adjacent")
	}
	c := Range{Start: 0x3000, End: 0x4000}
	if a.Adjacent(c) {
		t.Error("expected not adjacent")
	}
}
