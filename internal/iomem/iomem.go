// Package iomem provides parsing of /proc/iomem for physical RAM range discovery.
package iomem

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

// ErrPermissionDenied indicates that /proc/iomem returned zeroed ranges,
// typically because the caller lacks CAP_SYS_ADMIN.
var ErrPermissionDenied = errors.New("need CAP_SYS_ADMIN to read /proc/iomem")

// ErrParseLine indicates a malformed line in /proc/iomem.
var ErrParseLine = errors.New("unable to parse iomem line")

// ParseFile reads and parses the /proc/iomem file at the given path.
func ParseFile(path string) ([]phys.Range, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	ranges, err := Parse(file)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return ranges, nil
}

// Parse reads /proc/iomem content from an io.Reader and returns merged
// System RAM ranges. Only top-level lines (no leading whitespace) that end
// with " : System RAM" are included. Child/indented lines are ignored.
//
// If a System RAM range of 0-0 is found, ErrPermissionDenied is returned.
func Parse(reader io.Reader) ([]phys.Range, error) {
	var ranges []phys.Range

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip child/indented lines.
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		// Only keep top-level System RAM lines.
		const systemRAM = " : System RAM"
		if !strings.HasSuffix(line, systemRAM) {
			continue
		}

		// Extract the "START-END" part before the colon.
		rangePart := strings.TrimSuffix(line, systemRAM)
		if rangePart == "" {
			continue
		}

		startStr, endStr, ok := splitRange(rangePart)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrParseLine, line)
		}

		start, err := strconv.ParseUint(startStr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid start %q: %w", ErrParseLine, startStr, err)
		}

		end, err := strconv.ParseUint(endStr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid end %q: %w", ErrParseLine, endStr, err)
		}

		// Detect permission-denied /proc/iomem (zeroed ranges).
		if start == 0 && end == 0 {
			return nil, ErrPermissionDenied
		}

		if end == ^uint64(0) {
			return nil, fmt.Errorf("%w: end address overflows exclusive range: %s", ErrParseLine, line)
		}

		// /proc/iomem reports inclusive end addresses. GOFVML stores ranges
		// with exclusive ends so length, contains, and adjacency checks stay
		// consistent across physical-memory workflows.
		ranges = append(ranges, phys.Range{Start: start, End: end + 1})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read iomem: %w", err)
	}

	return Merge(ranges), nil
}

// splitRange splits a "START-END" string into its components.
func splitRange(s string) (start, end string, ok bool) {
	hyphen := strings.IndexByte(s, '-')
	if hyphen < 0 {
		return "", "", false
	}
	return s[:hyphen], s[hyphen+1:], true
}

// Merge sorts and merges overlapping or adjacent ranges.
func Merge(ranges []phys.Range) []phys.Range {
	if len(ranges) <= 1 {
		return ranges
	}

	// Sort by start address.
	result := make([]phys.Range, len(ranges))
	copy(result, ranges)

	// Simple bubble sort for determinism (or use sort.Slice).
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Start < result[i].Start {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// Merge overlapping and adjacent ranges.
	merged := make([]phys.Range, 0, len(result))
	current := result[0]

	for i := 1; i < len(result); i++ {
		next := result[i]
		// Overlap: next.Start <= current.End (AVML uses >= for start check)
		// Adjacent: next.Start == current.End
		if next.Start <= current.End {
			// Overlapping or adjacent - extend current if needed.
			if next.End > current.End {
				current.End = next.End
			}
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	return merged
}

// Split splits ranges into chunks of at most maxSize bytes.
func Split(ranges []phys.Range, maxSize uint64) []phys.Range {
	var result []phys.Range

	for _, r := range ranges {
		for r.Start < r.End {
			end := r.Start + maxSize
			if end > r.End {
				end = r.End
			}
			result = append(result, phys.Range{Start: r.Start, End: end})
			r.Start = end
		}
	}

	return result
}
