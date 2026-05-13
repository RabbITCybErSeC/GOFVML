// Package diskusage provides preflight disk usage estimation for memory acquisition.
package diskusage

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// Estimate returns the estimated disk space needed for the given physical
// memory ranges in bytes. For LiME format, the estimate is the sum of range
// lengths plus header overhead. For AVML-compressed, the estimate is a
// fraction of the total (compression varies by content).
func Estimate(ranges []struct{ Start, End uint64 }, format string) uint64 {
	var total uint64
	for _, r := range ranges {
		size := r.End - r.Start
		total += size
	}

	// Add header overhead: 32 bytes per range for LiME/AVML headers.
	headerOverhead := uint64(len(ranges) * 32)
	total += headerOverhead

	switch format {
	case "avml", "compressed":
		// AVML compression typically achieves 2-4x on memory images.
		// Use conservative 1.5x estimate (some blocks are zeros and omitted).
		return total / 3 * 2
	case "lime", "raw":
		// LiME and raw have no compression overhead beyond headers.
		return total
	default:
		return total
	}
}

// CheckAvailable checks if the directory for the given path has enough
// free space for the estimated size.
func CheckAvailable(path string, estimatedBytes uint64) (bool, uint64, error) {
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}

	var statfs syscall.Statfs_t
	if err := syscall.Statfs(dir, &statfs); err != nil {
		return false, 0, fmt.Errorf("statfs %s: %w", dir, err)
	}

	available := statfs.Bavail * uint64(statfs.Bsize)
	return available >= estimatedBytes, available, nil
}

// PreflightResult contains the outcome of a preflight check.
type PreflightResult struct {
	// EstimatedBytes is the estimated output size.
	EstimatedBytes uint64
	// AvailableBytes is the available disk space.
	AvailableBytes uint64
	// Sufficient is true if available space >= estimated space.
	Sufficient bool
	// Path is the output path that was checked.
	Path string
}

// Preflight checks if the output path has sufficient space for the acquisition.
func Preflight(path string, ranges []struct{ Start, End uint64 }, format string) (*PreflightResult, error) {
	estimated := Estimate(ranges, format)

	sufficient, available, err := CheckAvailable(path, estimated)
	if err != nil {
		return nil, err
	}

	return &PreflightResult{
		EstimatedBytes: estimated,
		AvailableBytes: available,
		Sufficient:     sufficient,
		Path:           path,
	}, nil
}
