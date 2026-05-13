package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/procfs"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

// DefaultChunkSize is the default chunk size for process memory reads.
const DefaultChunkSize = 1 * 1024 * 1024 // 1 MiB

// Options configures process memory acquisition.
type Options struct {
	// PID is the target process ID.
	PID int
	// Filter selects which mappings to read.
	Filter Filter
	// ChunkSize is the maximum bytes to read per chunk.
	// If zero, DefaultChunkSize is used.
	ChunkSize int
	// Strict, if true, fails the entire PID when any mapping cannot be fully read.
	Strict bool
	// Progress is an optional progress callback.
	Progress progress.Callback
}

// Result holds the outcome of process memory acquisition for one PID.
type Result struct {
	// Success is true if acquisition completed without fatal errors.
	Success bool
	// PID is the target process ID.
	PID int
	// BytesRead is the total bytes successfully read across all mappings.
	BytesRead uint64
	// Mappings holds per-mapping results.
	Mappings []MappingResult
	// Warnings contains non-fatal issues.
	Warnings []*diagnostic.Diagnostic
}

// MappingResult holds the outcome for a single mapping.
type MappingResult struct {
	// Mapping is the original mapping metadata.
	Mapping procfs.Mapping
	// BytesRead is the total bytes successfully read from this mapping.
	BytesRead uint64
	// Events records each read attempt.
	Events []ReadEvent
	// Blocks contains acquired payload chunks for this mapping.
	Blocks []PayloadBlock
}

// ReadEvent records a single read attempt.
type ReadEvent struct {
	// VirtualAddress is the virtual address where the read started.
	VirtualAddress uint64
	// Requested is the number of bytes requested.
	Requested int
	// Read is the number of bytes actually read.
	Read int
	// Err is non-nil if the read failed.
	Err error
}

// IsError reports whether this event represents a failed read.
func (e ReadEvent) IsError() bool {
	return e.Err != nil
}

// IsShortRead reports whether this event represents a short read.
func (e ReadEvent) IsShortRead() bool {
	return e.Err == nil && e.Read < e.Requested
}

// Acquire reads process memory for the given PID according to options.
// It returns a Result with per-mapping events and any warnings or errors.
func Acquire(ctx context.Context, opts Options) (*Result, error) {
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = DefaultChunkSize
	}

	result := &Result{
		PID: opts.PID,
	}

	// Open /proc/<pid>/mem read-only.
	memPath := filepath.Join("/proc", strconv.Itoa(opts.PID), "mem")
	memFile, err := os.OpenFile(memPath, os.O_RDONLY, 0)
	if err != nil {
		d := diagnostic.ProcessError("cannot open process memory").
			WithTarget(memPath).
			WithCause(err).
			WithSuggestion("ensure the process exists and you have ptrace access")
		return nil, d
	}
	defer memFile.Close()

	// Read process maps.
	maps, err := procfs.ReadMaps(opts.PID)
	if err != nil {
		d := diagnostic.ProcessError("cannot read process maps").
			WithTarget(strconv.Itoa(opts.PID)).
			WithCause(err)
		return nil, d
	}

	// Select mappings to read.
	selected := SelectMappings(maps, opts.Filter)
	if len(selected) == 0 {
		result.Warnings = append(result.Warnings,
			diagnostic.Warning(diagnostic.CategoryProcess, "no mappings selected for acquisition").
				WithTarget(strconv.Itoa(opts.PID)))
		result.Success = true
		return result, nil
	}

	reporter := progress.NewReporter(opts.Progress)
	cr := progress.NewContextReporter(ctx, reporter)

	totalMappings := uint64(len(selected))
	for i, mapping := range selected {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		mr := MappingResult{Mapping: mapping}
		mappingErr := readMapping(ctx, memFile, mapping, opts.ChunkSize, &mr)

		if mappingErr != nil {
			if opts.Strict {
				result.Success = false
				result.Mappings = append(result.Mappings, mr)
				return result, mappingErr
			}
			// Non-strict: record the error as a warning and continue.
			result.Warnings = append(result.Warnings,
				diagnostic.Warning(diagnostic.CategoryProcess, "mapping read failed").
					WithTarget(fmt.Sprintf("pid=%d mapping=%x-%x", opts.PID, mapping.Start, mapping.End)).
					WithCause(mappingErr))
		}

		result.Mappings = append(result.Mappings, mr)
		result.BytesRead += mr.BytesRead

		_ = cr.Report(progress.Event{
			Operation: "process acquisition",
			Phase:     "reading",
			Current:   uint64(i + 1),
			Total:     totalMappings,
			Target:    fmt.Sprintf("pid=%d", opts.PID),
		})
	}

	result.Success = true
	return result, nil
}

// readMapping reads all chunks from a single mapping.
func readMapping(ctx context.Context, memFile *os.File, mapping procfs.Mapping, chunkSize int, mr *MappingResult) error {
	addr := mapping.Start
	end := mapping.End

	for addr < end {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		remaining := end - addr
		size := uint64(chunkSize)
		if remaining < size {
			size = remaining
		}

		buf := make([]byte, size)
		n, err := memFile.ReadAt(buf, int64(addr))
		status := StatusOK
		if err != nil {
			status = StatusError
		} else if n < int(size) {
			status = StatusShortRead
		}

		mr.Events = append(mr.Events, ReadEvent{
			VirtualAddress: addr,
			Requested:      int(size),
			Read:           n,
			Err:            err,
		})
		if n > 0 {
			mr.BytesRead += uint64(n)
			mr.Blocks = append(mr.Blocks, PayloadBlock{
				VirtualAddress:  addr,
				CompressionType: CompressionNone,
				Status:          status,
				Data:            append([]byte(nil), buf[:n]...),
			})
		}
		if err != nil {
			return fmt.Errorf("read mapping chunk at 0x%x: %w", addr, err)
		}
		if n < int(size) {
			return fmt.Errorf("short read mapping chunk at 0x%x: %w", addr, io.ErrUnexpectedEOF)
		}

		// Always advance by the full chunk size to avoid getting stuck
		// on unmapped pages within the mapping.  If the read succeeded
		// and returned fewer bytes than requested, the skipped region
		// is unmapped and retrying would yield the same short read.
		addr += size
	}

	return nil
}
