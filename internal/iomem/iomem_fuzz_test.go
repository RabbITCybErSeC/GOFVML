package iomem

import (
	"strings"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

func FuzzParse(f *testing.F) {
	// Seed corpus.
	f.Add("00000000-00000000 : System RAM\n")
	f.Add("00001000-00002000 : System RAM\n")
	f.Add("  00001000-00002000 : System RAM\n") // indented
	f.Add("00001000-00002000 : Reserved\n")     // not System RAM
	f.Add("malformed line\n")
	f.Add("")
	f.Add("00000000-00000000 : System RAM\n") // zeroed range

	f.Fuzz(func(t *testing.T, data string) {
		// Parse should not panic on arbitrary input.
		reader := strings.NewReader(data)
		_, _ = Parse(reader)
	})
}

func TestParse_MalformedLines(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "missing hyphen",
			input:   "0000100000002000 : System RAM\n",
			wantErr: true,
		},
		{
			name:    "invalid start",
			input:   "zzzz-00002000 : System RAM\n",
			wantErr: true,
		},
		{
			name:    "invalid end",
			input:   "00001000-zzzz : System RAM\n",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false, // empty input returns empty ranges, not error
		},
		{
			name:    "no system ram",
			input:   "00001000-00002000 : Reserved\n",
			wantErr: false, // no error, just no ranges
		},
		{
			name:    "zeroed range",
			input:   "00000000-00000000 : System RAM\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := Parse(reader)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMerge_OverlapAndAdjacency(t *testing.T) {
	tests := []struct {
		name   string
		input  []phys.Range
		want   []phys.Range
	}{
		{
			name:   "adjacent ranges",
			input:  []phys.Range{{Start: 0, End: 0x1000}, {Start: 0x1000, End: 0x2000}},
			want:   []phys.Range{{Start: 0, End: 0x2000}},
		},
		{
			name:   "overlapping ranges",
			input:  []phys.Range{{Start: 0, End: 0x1500}, {Start: 0x1000, End: 0x2000}},
			want:   []phys.Range{{Start: 0, End: 0x2000}},
		},
		{
			name:   "disjoint ranges",
			input:  []phys.Range{{Start: 0, End: 0x1000}, {Start: 0x2000, End: 0x3000}},
			want:   []phys.Range{{Start: 0, End: 0x1000}, {Start: 0x2000, End: 0x3000}},
		},
		{
			name:   "empty input",
			input:  []phys.Range{},
			want:   []phys.Range{},
		},
		{
			name:   "single range",
			input:  []phys.Range{{Start: 0, End: 0x1000}},
			want:   []phys.Range{{Start: 0, End: 0x1000}},
		},
		{
			name:   "unsorted ranges",
			input:  []phys.Range{{Start: 0x2000, End: 0x3000}, {Start: 0, End: 0x1000}},
			want:   []phys.Range{{Start: 0, End: 0x1000}, {Start: 0x2000, End: 0x3000}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d ranges, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("range %d: expected %+v, got %+v", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestSplit_MaxSize(t *testing.T) {
	tests := []struct {
		name    string
		input   []phys.Range
		maxSize uint64
		want    []phys.Range
	}{
		{
			name:    "no split needed",
			input:   []phys.Range{{Start: 0, End: 0x1000}},
			maxSize: 0x2000,
			want:    []phys.Range{{Start: 0, End: 0x1000}},
		},
		{
			name:    "split in half",
			input:   []phys.Range{{Start: 0, End: 0x2000}},
			maxSize: 0x1000,
			want:    []phys.Range{{Start: 0, End: 0x1000}, {Start: 0x1000, End: 0x2000}},
		},
		{
			name:    "multiple ranges",
			input:   []phys.Range{{Start: 0, End: 0x2000}, {Start: 0x10000, End: 0x12000}},
			maxSize: 0x1000,
			want: []phys.Range{
				{Start: 0, End: 0x1000},
				{Start: 0x1000, End: 0x2000},
				{Start: 0x10000, End: 0x11000},
				{Start: 0x11000, End: 0x12000},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Split(tt.input, tt.maxSize)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d ranges, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("range %d: expected %+v, got %+v", i, tt.want[i], got[i])
				}
			}
		})
	}
}
