package iomem

import (
	"bytes"
	"errors"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

func TestParseFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected []phys.Range
	}{
		{
			name:     "iomem.txt",
			filename: "../../fixtures/iomem/iomem.txt",
			expected: []phys.Range{
				iomemRange(4096, 654335),
				iomemRange(1048576, 1073676287),
				iomemRange(4294967296, 6979321855),
			},
		},
		{
			name:     "iomem-2.txt",
			filename: "../../fixtures/iomem/iomem-2.txt",
			expected: []phys.Range{
				iomemRange(4096, 655359),
				iomemRange(1048576, 1055838207),
				iomemRange(1056026624, 1073328127),
				iomemRange(1073737728, 1073741823),
				iomemRange(4294967296, 6979321855),
			},
		},
		{
			name:     "iomem-3.txt",
			filename: "../../fixtures/iomem/iomem-3.txt",
			expected: []phys.Range{
				iomemRange(65536, 649215),
				iomemRange(1048576, 2146303999),
				iomemRange(2146435072, 2147483647),
			},
		},
		{
			name:     "iomem-4.txt",
			filename: "../../fixtures/iomem/iomem-4.txt",
			expected: []phys.Range{
				iomemRange(4096, 655359),
				iomemRange(1048576, 1423523839),
				iomemRange(1423585280, 1511186431),
				iomemRange(1780150272, 1818623999),
				iomemRange(1818828800, 1843613695),
				iomemRange(2071535616, 2071986175),
				iomemRange(4294967296, 414464344063),
			},
		},
		{
			name:     "iomem-5.txt",
			filename: "../../fixtures/iomem/iomem-5.txt",
			expected: []phys.Range{
				iomemRange(4096, 655359),
				iomemRange(1048576, 241524735),
				iomemRange(241643520, 251310079),
				iomemRange(251326464, 251383807),
				iomemRange(251424768, 264671231),
				iomemRange(264675328, 267280383),
				iomemRange(267739136, 267866111),
				iomemRange(267870208, 3221225471),
				iomemRange(4294967296, 13958643711),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranges, err := ParseFile(tt.filename)
			if err != nil {
				t.Fatalf("ParseFile(%q) error: %v", tt.filename, err)
			}

			if len(ranges) != len(tt.expected) {
				t.Fatalf("got %d ranges, want %d", len(ranges), len(tt.expected))
			}

			for i, got := range ranges {
				want := tt.expected[i]
				if got != want {
					t.Errorf("range[%d] = {Start: %d, End: %d}, want {Start: %d, End: %d}",
						i, got.Start, got.End, want.Start, want.End)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []phys.Range
		wantErr  error
	}{
		{
			name:     "empty",
			input:    "",
			expected: nil,
		},
		{
			name:     "no_system_ram",
			input:    "00000000-00000fff : reserved\n",
			expected: nil,
		},
		{
			name:  "single_range",
			input: "00100000-3ffeffff : System RAM\n",
			expected: []phys.Range{
				iomemRange(0x100000, 0x3ffeffff),
			},
		},
		{
			name: "child_lines_ignored",
			input: "00100000-3ffeffff : System RAM\n" +
				"  01000000-01825b20 : Kernel code\n",
			expected: []phys.Range{
				iomemRange(0x100000, 0x3ffeffff),
			},
		},
		{
			name:     "permission_denied",
			input:    "00000000-00000000 : System RAM\n",
			expected: nil,
			wantErr:  ErrPermissionDenied,
		},
		{
			name: "multiple_ranges_merged",
			input: "00001000-0009ffff : System RAM\n" +
				"00100000-3ffeffff : System RAM\n",
			expected: []phys.Range{
				iomemRange(0x1000, 0x9ffff),
				iomemRange(0x100000, 0x3ffeffff),
			},
		},
		{
			name: "inclusive_iomem_ranges_merge_when_exclusive_ranges_touch",
			input: "00001000-00001fff : System RAM\n" +
				"00002000-00002fff : System RAM\n",
			expected: []phys.Range{
				{Start: 0x1000, End: 0x3000},
			},
		},
		{
			name:     "malformed_line_no_hyphen",
			input:    "00100000 : System RAM\n",
			expected: nil,
			wantErr:  ErrParseLine,
		},
		{
			name:     "malformed_line_bad_hex",
			input:    "00100000-zzzzzzzz : System RAM\n",
			expected: nil,
			wantErr:  ErrParseLine,
		},
		{
			name:     "exclusive_end_overflow",
			input:    "ffffffffffffffff-ffffffffffffffff : System RAM\n",
			expected: nil,
			wantErr:  ErrParseLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ranges, err := Parse(bytes.NewReader([]byte(tt.input)))

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(ranges) != len(tt.expected) {
				t.Fatalf("got %d ranges, want %d", len(ranges), len(tt.expected))
			}

			for i, got := range ranges {
				want := tt.expected[i]
				if got != want {
					t.Errorf("range[%d] = {Start: %d, End: %d}, want {Start: %d, End: %d}",
						i, got.Start, got.End, want.Start, want.End)
				}
			}
		})
	}
}

func iomemRange(start, inclusiveEnd uint64) phys.Range {
	return phys.Range{Start: start, End: inclusiveEnd + 1}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		input    []phys.Range
		expected []phys.Range
	}{
		{
			name:     "empty",
			input:    []phys.Range{},
			expected: []phys.Range{},
		},
		{
			name: "single",
			input: []phys.Range{
				{Start: 0, End: 10},
			},
			expected: []phys.Range{
				{Start: 0, End: 10},
			},
		},
		{
			name: "adjacent",
			input: []phys.Range{
				{Start: 0, End: 3},
				{Start: 3, End: 6},
				{Start: 6, End: 10},
			},
			expected: []phys.Range{
				{Start: 0, End: 10},
			},
		},
		{
			name: "overlapping",
			input: []phys.Range{
				{Start: 0, End: 5},
				{Start: 3, End: 8},
				{Start: 7, End: 10},
			},
			expected: []phys.Range{
				{Start: 0, End: 10},
			},
		},
		{
			name: "non_overlapping",
			input: []phys.Range{
				{Start: 0, End: 3},
				{Start: 3, End: 6},
				{Start: 7, End: 10},
				{Start: 12, End: 15},
			},
			expected: []phys.Range{
				{Start: 0, End: 6},
				{Start: 7, End: 10},
				{Start: 12, End: 15},
			},
		},
		{
			name: "unsorted",
			input: []phys.Range{
				{Start: 7, End: 10},
				{Start: 0, End: 3},
				{Start: 3, End: 6},
			},
			expected: []phys.Range{
				{Start: 0, End: 6},
				{Start: 7, End: 10},
			},
		},
		{
			name: "contained",
			input: []phys.Range{
				{Start: 0, End: 10},
				{Start: 2, End: 5},
				{Start: 7, End: 8},
			},
			expected: []phys.Range{
				{Start: 0, End: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.input)

			if len(got) != len(tt.expected) {
				t.Fatalf("got %d ranges, want %d", len(got), len(tt.expected))
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("range[%d] = {Start: %d, End: %d}, want {Start: %d, End: %d}",
						i, got[i].Start, got[i].End, tt.expected[i].Start, tt.expected[i].End)
				}
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    []phys.Range
		maxSize  uint64
		expected []phys.Range
	}{
		{
			name:     "empty",
			input:    []phys.Range{},
			maxSize:  10,
			expected: nil,
		},
		{
			name: "exact_split",
			input: []phys.Range{
				{Start: 0, End: 30},
			},
			maxSize: 10,
			expected: []phys.Range{
				{Start: 0, End: 10},
				{Start: 10, End: 20},
				{Start: 20, End: 30},
			},
		},
		{
			name: "uneven_split",
			input: []phys.Range{
				{Start: 0, End: 30},
			},
			maxSize: 7,
			expected: []phys.Range{
				{Start: 0, End: 7},
				{Start: 7, End: 14},
				{Start: 14, End: 21},
				{Start: 21, End: 28},
				{Start: 28, End: 30},
			},
		},
		{
			name: "multiple_ranges",
			input: []phys.Range{
				{Start: 0, End: 10},
				{Start: 10, End: 20},
				{Start: 20, End: 30},
			},
			maxSize: 7,
			expected: []phys.Range{
				{Start: 0, End: 7},
				{Start: 7, End: 10},
				{Start: 10, End: 17},
				{Start: 17, End: 20},
				{Start: 20, End: 27},
				{Start: 27, End: 30},
			},
		},
		{
			name: "no_split_needed",
			input: []phys.Range{
				{Start: 0, End: 5},
			},
			maxSize: 10,
			expected: []phys.Range{
				{Start: 0, End: 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Split(tt.input, tt.maxSize)

			if len(got) != len(tt.expected) {
				t.Fatalf("got %d ranges, want %d", len(got), len(tt.expected))
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("range[%d] = {Start: %d, End: %d}, want {Start: %d, End: %d}",
						i, got[i].Start, got[i].End, tt.expected[i].Start, tt.expected[i].End)
				}
			}
		})
	}
}
