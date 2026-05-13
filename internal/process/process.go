// Package process provides PID-scoped process memory acquisition.
package process

import (
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
)

// Filter selects which mappings to acquire.
type Filter struct {
	// RequireReadable, if true, selects only mappings with read permission.
	RequireReadable bool
	// MinAddress, if non-zero, selects mappings that overlap [MinAddress, MaxAddress).
	MinAddress uint64
	// MaxAddress, if non-zero, selects mappings that overlap [MinAddress, MaxAddress).
	MaxAddress uint64
	// PathnameMatch, if non-empty, selects mappings whose pathname contains this substring.
	PathnameMatch string
	// MaxBytes limits the total bytes selected across all mappings.
	// If zero, no byte limit is applied.
	MaxBytes uint64
}

// DefaultFilter returns a filter that selects readable mappings with no other restrictions.
func DefaultFilter() Filter {
	return Filter{RequireReadable: true}
}

// SelectMappings returns the subset of mappings that match the filter.
// If MaxBytes is set, selection stops once the total selected bytes would exceed the limit.
func SelectMappings(mappings []procfs.Mapping, filter Filter) []procfs.Mapping {
	var selected []procfs.Mapping
	var totalBytes uint64

	for _, m := range mappings {
		if filter.RequireReadable && !m.IsReadable() {
			continue
		}

		if filter.MinAddress != 0 || filter.MaxAddress != 0 {
			if !overlapsRange(m, filter.MinAddress, filter.MaxAddress) {
				continue
			}
		}

		if filter.PathnameMatch != "" && !strings.Contains(m.Pathname, filter.PathnameMatch) {
			continue
		}

		if filter.MaxBytes > 0 {
			length := m.Len()
			if totalBytes+length > filter.MaxBytes {
				// Partially include if there's remaining budget.
				remaining := filter.MaxBytes - totalBytes
				if remaining == 0 {
					break
				}
				// Truncate the mapping to fit.
				m = truncateMapping(m, remaining)
				selected = append(selected, m)
				break
			}
			totalBytes += length
		}

		selected = append(selected, m)
	}

	return selected
}

// overlapsRange reports whether m overlaps [min, max).
// If max is zero, it is treated as no upper bound.
func overlapsRange(m procfs.Mapping, min, max uint64) bool {
	if max > 0 && m.Start >= max {
		return false
	}
	if m.End <= min {
		return false
	}
	return true
}

// truncateMapping returns a copy of m with End adjusted so that Len() <= maxLen.
func truncateMapping(m procfs.Mapping, maxLen uint64) procfs.Mapping {
	if m.Len() <= maxLen {
		return m
	}
	m.End = m.Start + maxLen
	return m
}
