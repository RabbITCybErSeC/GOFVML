package diskusage

import (
	"testing"
)

func TestEstimate(t *testing.T) {
	ranges := []struct{ Start, End uint64 }{
		{0x1000, 0x2000},
		{0x3000, 0x5000},
	}

	// Total raw size: 0x1000 + 0x2000 = 0x3000 = 12288 bytes
	// Plus header overhead: 2 * 32 = 64 bytes
	// Total: 12352 bytes

	tests := []struct {
		format string
		min    uint64
		max    uint64
	}{
		{"raw", 12352, 12352},
		{"lime", 12352, 12352},
		{"avml", 8000, 12352}, // compressed estimate should be less
		{"unknown", 12352, 12352},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			est := Estimate(ranges, tt.format)
			if est < tt.min || est > tt.max {
				t.Errorf("Estimate(%q) = %d, want between %d and %d", tt.format, est, tt.min, tt.max)
			}
		})
	}
}

func TestEstimateEmpty(t *testing.T) {
	ranges := []struct{ Start, End uint64 }{}
	est := Estimate(ranges, "raw")
	if est != 0 {
		t.Errorf("Estimate(empty) = %d, want 0", est)
	}
}

func TestPreflight(t *testing.T) {
	ranges := []struct{ Start, End uint64 }{
		{0, 1024},
	}

	result, err := Preflight("/tmp/test.lime", ranges, "lime")
	if err != nil {
		// This may fail on systems where /tmp doesn't exist or statfs fails.
		t.Skipf("Preflight failed (may be expected on this platform): %v", err)
	}

	if result.Path != "/tmp/test.lime" {
		t.Errorf("Path = %q, want /tmp/test.lime", result.Path)
	}
	if result.EstimatedBytes == 0 {
		t.Error("Expected non-zero estimated bytes")
	}
	if result.AvailableBytes == 0 {
		t.Error("Expected non-zero available bytes")
	}
}
