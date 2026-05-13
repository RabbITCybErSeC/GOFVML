// Package source provides memory source detection and read adapters.
package source

import (
	"context"
	"fmt"
	"io"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

// Reader is the interface for reading physical memory from a source.
type Reader interface {
	// ReadAt reads up to len(p) bytes from the source at the given physical
	// offset. It returns the number of bytes read and any error encountered.
	ReadAt(p []byte, off uint64) (int, error)
	io.Closer
}

// Availability indicates whether a source is available and why it might not be.
type Availability struct {
	// Available is true if the source can be used.
	Available bool
	// Path is the device or file path for the source.
	Path string
	// Reason is a human-readable explanation of availability status.
	Reason string
	// Diagnostic is set when the source is not available due to an error.
	Diagnostic *diagnostic.Diagnostic
}

// Info describes a memory source without opening it.
type Info struct {
	// Name is the source identifier (e.g., "crash", "kcore", "mem").
	Name string
	// Path is the device or file path.
	Path string
	// Priority is the fallback priority (lower is preferred).
	Priority int
}

// Source is a physical memory source that can be probed and opened.
type Source interface {
	// Info returns static information about the source.
	Info() Info
	// Check returns the current availability of the source.
	Check(ctx context.Context) Availability
	// Open opens the source for reading. Returns a diagnostic if the source
	// cannot be opened.
	Open(ctx context.Context) (Reader, *diagnostic.Diagnostic)
}

// ReadBlock reads a single physical block from a reader.
func ReadBlock(r Reader, block phys.Block) (phys.Block, error) {
	if block.Range.IsEmpty() {
		return block, nil
	}

	data := make([]byte, block.Range.Len())
	n, err := r.ReadAt(data, block.Range.Start)
	if err != nil {
		return phys.Block{}, fmt.Errorf("read block at 0x%x: %w", block.Range.Start, err)
	}

	return phys.Block{
		Range: block.Range,
		Data:  data[:n],
	}, nil
}

// ReadBlocks reads multiple physical blocks from a reader.
func ReadBlocks(ctx context.Context, r Reader, blocks []phys.Block) ([]phys.Block, []error) {
	results := make([]phys.Block, 0, len(blocks))
	errors := make([]error, 0, len(blocks))

	for _, block := range blocks {
		select {
		case <-ctx.Done():
			results = append(results, phys.Block{})
			errors = append(errors, ctx.Err())
			return results, errors
		default:
		}

		readBlock, err := ReadBlock(r, block)
		if err != nil {
			results = append(results, phys.Block{})
			errors = append(errors, err)
		} else {
			results = append(results, readBlock)
			errors = append(errors, nil)
		}
	}

	return results, errors
}

// IsAllZeros reports whether data contains only zero bytes.
func IsAllZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// Common source names and paths.
const (
	SourceCrash  = "crash"
	SourceKcore  = "kcore"
	SourceMem    = "mem"
	PathDevCrash = "/dev/crash"
	PathProcKcore = "/proc/kcore"
	PathDevMem    = "/dev/mem"
)

// DefaultSources returns the default fallback-ordered source list.
func DefaultSources() []Source {
	return []Source{
		NewDevCrashSource(),
		NewProcKcoreSource(),
		NewDevMemSource(),
	}
}

// FindAvailableSource returns the first available source from the given list.
func FindAvailableSource(ctx context.Context, sources []Source) (Source, Availability) {
	for _, src := range sources {
		avail := src.Check(ctx)
		if avail.Available {
			return src, avail
		}
	}
	return nil, Availability{
		Available: false,
		Reason:    "no available memory sources found",
	}
}

// OpenExplicit opens a specific source by name from the given list.
// Returns an error if the source is not found or not available.
func OpenExplicit(ctx context.Context, sources []Source, name string) (Reader, *diagnostic.Diagnostic) {
	for _, src := range sources {
		if src.Info().Name == name {
			avail := src.Check(ctx)
			if !avail.Available {
				return nil, diagnostic.SourceError("source not available").
					WithOperation("physical acquisition").
					WithTarget(src.Info().Path).
					WithCause(fmt.Errorf("%s", avail.Reason))
			}
			return src.Open(ctx)
		}
	}
	return nil, diagnostic.SourceError("unknown source").
		WithOperation("physical acquisition").
		WithTarget(name).
		WithSuggestion("valid sources: crash, kcore, mem")
}
