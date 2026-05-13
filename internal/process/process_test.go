package process

import (
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
)

func TestDefaultFilter(t *testing.T) {
	f := DefaultFilter()
	if !f.RequireReadable {
		t.Error("expected RequireReadable to be true")
	}
}

func TestSelectMappings(t *testing.T) {
	mappings := []procfs.Mapping{
		{Start: 0x1000, End: 0x2000, Perms: "r--p", Pathname: "/lib/foo.so"},
		{Start: 0x2000, End: 0x3000, Perms: "r-xp", Pathname: "/lib/foo.so"},
		{Start: 0x3000, End: 0x4000, Perms: "rw-p", Pathname: "[heap]"},
		{Start: 0x4000, End: 0x5000, Perms: "---p", Pathname: ""},
		{Start: 0x5000, End: 0x6000, Perms: "r--p", Pathname: "/lib/bar.so"},
	}

	tests := []struct {
		name   string
		filter Filter
		want   int
		check  func([]procfs.Mapping) bool
	}{
		{
			name:   "default readable only",
			filter: DefaultFilter(),
			want:   4,
		},
		{
			name:   "no filter",
			filter: Filter{},
			want:   5,
		},
		{
			name: "address range filter",
			filter: Filter{
				RequireReadable: true,
				MinAddress:      0x1500,
				MaxAddress:      0x5500,
			},
			want: 4,
			check: func(m []procfs.Mapping) bool {
				// Should include 0x1000-0x2000 (overlaps), 0x2000-0x3000, 0x3000-0x4000, 0x5000-0x6000 (overlaps)
				return len(m) == 4
			},
		},
		{
			name: "pathname match",
			filter: Filter{
				RequireReadable: true,
				PathnameMatch:   "bar",
			},
			want: 1,
			check: func(m []procfs.Mapping) bool {
				return len(m) == 1 && m[0].Pathname == "/lib/bar.so"
			},
		},
		{
			name: "max bytes",
			filter: Filter{
				RequireReadable: true,
				MaxBytes:        0x1800,
			},
			want: 2,
			check: func(m []procfs.Mapping) bool {
				// First mapping is 0x1000 bytes, second is 0x1000 bytes
				// Total would be 0x2000, but max is 0x1800.
				// So first full (0x1000), second truncated (0x800).
				if len(m) != 2 {
					return false
				}
				if m[0].Len() != 0x1000 {
					return false
				}
				if m[1].Len() != 0x800 {
					return false
				}
				return true
			},
		},
		{
			name: "max bytes exact",
			filter: Filter{
				RequireReadable: true,
				MaxBytes:        0x1000,
			},
			want: 1,
			check: func(m []procfs.Mapping) bool {
				return len(m) == 1 && m[0].Len() == 0x1000
			},
		},
		{
			name: "max bytes zero",
			filter: Filter{
				RequireReadable: true,
				MaxBytes:        0,
			},
			want: 4,
		},
		{
			name: "non-readable excluded",
			filter: Filter{
				RequireReadable: true,
			},
			want: 4,
			check: func(m []procfs.Mapping) bool {
				for _, mm := range m {
					if !mm.IsReadable() {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectMappings(mappings, tt.filter)
			if len(got) != tt.want {
				t.Errorf("SelectMappings() returned %d mappings, want %d", len(got), tt.want)
			}
			if tt.check != nil && !tt.check(got) {
				t.Errorf("SelectMappings() check failed for %+v", got)
			}
		})
	}
}

func TestSelectMappingsRangeEdgeCases(t *testing.T) {
	mappings := []procfs.Mapping{
		{Start: 0x1000, End: 0x2000, Perms: "r--p"},
		{Start: 0x2000, End: 0x3000, Perms: "r--p"},
		{Start: 0x3000, End: 0x4000, Perms: "r--p"},
	}

	// Range that excludes first mapping
	f := Filter{RequireReadable: true, MinAddress: 0x2000, MaxAddress: 0x3500}
	got := SelectMappings(mappings, f)
	if len(got) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(got))
	}
	if got[0].Start != 0x2000 {
		t.Errorf("first mapping start = %x, want %x", got[0].Start, 0x2000)
	}
	if got[1].Start != 0x3000 {
		t.Errorf("second mapping start = %x, want %x", got[1].Start, 0x3000)
	}

	// Range that excludes all
	f2 := Filter{RequireReadable: true, MinAddress: 0x5000, MaxAddress: 0x6000}
	got2 := SelectMappings(mappings, f2)
	if len(got2) != 0 {
		t.Fatalf("expected 0 mappings, got %d", len(got2))
	}

	// No upper bound
	f3 := Filter{RequireReadable: true, MinAddress: 0x2500}
	got3 := SelectMappings(mappings, f3)
	if len(got3) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(got3))
	}
}

func TestSelectMappingsMaxBytesPartial(t *testing.T) {
	mappings := []procfs.Mapping{
		{Start: 0x1000, End: 0x3000, Perms: "r--p"}, // 0x2000 bytes
		{Start: 0x3000, End: 0x5000, Perms: "r--p"}, // 0x2000 bytes
	}

	f := Filter{RequireReadable: true, MaxBytes: 0x2800}
	got := SelectMappings(mappings, f)
	if len(got) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(got))
	}
	if got[0].Len() != 0x2000 {
		t.Errorf("first mapping len = %x, want %x", got[0].Len(), 0x2000)
	}
	if got[1].Len() != 0x800 {
		t.Errorf("second mapping len = %x, want %x", got[1].Len(), 0x800)
	}
}

func TestTruncateMapping(t *testing.T) {
	m := procfs.Mapping{Start: 0x1000, End: 0x3000}
	truncated := truncateMapping(m, 0x800)
	if truncated.Len() != 0x800 {
		t.Errorf("truncated len = %x, want %x", truncated.Len(), 0x800)
	}
	if truncated.Start != 0x1000 {
		t.Errorf("truncated start = %x, want %x", truncated.Start, 0x1000)
	}
	if truncated.End != 0x1800 {
		t.Errorf("truncated end = %x, want %x", truncated.End, 0x1800)
	}

	// No truncation needed
	m2 := procfs.Mapping{Start: 0x1000, End: 0x1800}
	truncated2 := truncateMapping(m2, 0x1000)
	if truncated2.Len() != 0x800 {
		t.Errorf("truncated2 len = %x, want %x", truncated2.Len(), 0x800)
	}
}
