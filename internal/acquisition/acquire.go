// Package acquisition provides the physical memory acquisition workflow.
package acquisition

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/image"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
	"github.com/RabbITCybErSeC/gofvml/internal/source"
)

// Options configures a physical acquisition operation.
type Options struct {
	// OutputPath is the destination file path for the acquired image.
	OutputPath string
	// Format is the output format: "lime" or "avml".
	Format string
	// SourceName is the explicit source to use (e.g., "crash", "kcore", "mem").
	// If empty, automatic fallback is used.
	SourceName string
	// Ranges are the physical memory ranges to acquire.
	Ranges []phys.Range
	// Progress is an optional progress callback.
	Progress progress.Callback
	// SkipZeroBlocks omits all-zero blocks from output when true.
	SkipZeroBlocks bool
}

// Result holds the outcome of a physical acquisition.
type Result struct {
	// Success is true if acquisition completed without fatal errors.
	Success bool
	// BytesWritten is the number of bytes written to the output file.
	BytesWritten uint64
	// BlocksWritten is the number of blocks written.
	BlocksWritten uint64
	// BlocksSkipped is the number of zero blocks skipped.
	BlocksSkipped uint64
	// Warnings contains non-fatal issues.
	Warnings []*diagnostic.Diagnostic
	// SourceName is the source that was used.
	SourceName string
	// OutputPath is the path to the output file.
	OutputPath string
}

// AddWarning appends a warning to the result.
func (r *Result) AddWarning(w *diagnostic.Diagnostic) {
	r.Warnings = append(r.Warnings, w)
}

// Acquire performs physical memory acquisition according to the given options.
func Acquire(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{
		OutputPath: opts.OutputPath,
	}

	// Validate options.
	if opts.OutputPath == "" {
		return nil, diagnostic.SourceError("output path is required").
			WithOperation("physical acquisition")
	}
	if len(opts.Ranges) == 0 {
		return nil, diagnostic.SourceError("no memory ranges specified").
			WithOperation("physical acquisition")
	}
	if opts.Format != "lime" && opts.Format != "avml" {
		return nil, diagnostic.FormatError("unsupported format").
			WithOperation("physical acquisition").
			WithTarget(opts.Format).
			WithSuggestion("use 'lime' or 'avml'")
	}

	// Open memory source.
	var reader source.Reader
	var diag *diagnostic.Diagnostic

	sources := source.DefaultSources()
	if opts.SourceName != "" {
		// Explicit source mode.
		reader, diag = source.OpenExplicit(ctx, sources, opts.SourceName)
		if diag != nil {
			return nil, diag
		}
		result.SourceName = opts.SourceName
	} else {
		// Auto fallback mode.
		var src source.Source
		var avail source.Availability
		src, avail = source.FindAvailableSource(ctx, sources)
		if src == nil {
			return nil, diagnostic.SourceError("no available memory sources").
				WithOperation("physical acquisition").
				WithCause(fmt.Errorf("%s", avail.Reason))
		}
		result.SourceName = src.Info().Name
		reader, diag = src.Open(ctx)
		if diag != nil {
			return nil, diag
		}
	}
	defer reader.Close()

	// Create output file safely.
	output, err := createSafeOutput(opts.OutputPath)
	if err != nil {
		return nil, diagnostic.SourceError("failed to create output file").
			WithOperation("physical acquisition").
			WithTarget(opts.OutputPath).
			WithCause(err)
	}
	defer output.Close()

	// Set up progress reporter.
	reporter := progress.NewReporter(opts.Progress)
	defer reporter.Close()

	// Calculate total bytes for progress.
	var totalBytes uint64
	for _, r := range opts.Ranges {
		totalBytes += r.Len()
	}

	// Acquire blocks.
	var bytesWritten uint64
	var blocksWritten uint64
	var blocksSkipped uint64
	var currentBytes uint64

	for _, rng := range opts.Ranges {
		select {
		case <-ctx.Done():
			result.Success = false
			result.BytesWritten = bytesWritten
			result.BlocksWritten = blocksWritten
			result.BlocksSkipped = blocksSkipped
			return result, ctx.Err()
		default:
		}

		// Read block from source.
		block := phys.Block{Range: rng}
		readBlock, err := source.ReadBlock(reader, block)
		if err != nil {
			result.AddWarning(diagnostic.Warning(diagnostic.CategorySource, "block read failed").
				WithOperation("physical acquisition").
				WithTarget(fmt.Sprintf("0x%x-0x%x", rng.Start, rng.End)).
				WithCause(err))
			continue
		}

		// Check for all-zero block.
		if opts.SkipZeroBlocks && readBlock.IsZero() {
			blocksSkipped++
			currentBytes += rng.Len()
			reporter.Report(progress.Event{
				Operation: "physical acquisition",
				Phase:     "reading",
				Target:    fmt.Sprintf("0x%x-0x%x", rng.Start, rng.End),
				Current:   currentBytes,
				Total:     totalBytes,
			})
			continue
		}

		// Write block to output.
		var written uint64
		var writeErr error
		switch opts.Format {
		case "lime":
			written, writeErr = writeLiMEBlock(output, readBlock)
		case "avml":
			written, writeErr = writeAVMLBlock(output, readBlock)
		}

		if writeErr != nil {
			result.AddWarning(diagnostic.Warning(diagnostic.CategoryFormat, "block write failed").
				WithOperation("physical acquisition").
				WithTarget(fmt.Sprintf("0x%x-0x%x", rng.Start, rng.End)).
				WithCause(writeErr))
			continue
		}

		bytesWritten += written
		blocksWritten++
		currentBytes += rng.Len()

		reporter.Report(progress.Event{
			Operation: "physical acquisition",
			Phase:     "writing",
			Target:    fmt.Sprintf("0x%x-0x%x", rng.Start, rng.End),
			Current:   currentBytes,
			Total:     totalBytes,
		})
	}

	result.Success = true
	result.BytesWritten = bytesWritten
	result.BlocksWritten = blocksWritten
	result.BlocksSkipped = blocksSkipped

	return result, nil
}

// createSafeOutput creates the output file with safe permissions.
// It uses O_EXCL to avoid following symlinks and sets 0600 permissions.
func createSafeOutput(path string) (*os.File, error) {
	// Create parent directories if needed.
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("create output directory: %w", err)
		}
	}

	// Open with O_EXCL to avoid following symlinks, O_CREATE to create if needed.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("create output file: %w", err)
	}

	return file, nil
}

// writeLiMEBlock writes a physical block as a LiME block.
func writeLiMEBlock(w io.Writer, block phys.Block) (uint64, error) {
	header := image.NewLiMEHeader(block.Range.Start, block.Range.End)
	if err := header.Encode(w); err != nil {
		return 0, fmt.Errorf("encode LiME header: %w", err)
	}

	if _, err := w.Write(block.Data); err != nil {
		return 0, fmt.Errorf("write LiME payload: %w", err)
	}

	return uint64(image.LiMEHeaderSize + len(block.Data)), nil
}

// writeAVMLBlock writes a physical block as an AVML-compressed block.
func writeAVMLBlock(w io.Writer, block phys.Block) (uint64, error) {
	header := image.NewAVMLHeader(block.Range.Start, block.Range.End)
	if err := image.EncodeAVMLBlock(w, header, block.Data); err != nil {
		return 0, fmt.Errorf("encode AVML block: %w", err)
	}

	// We can't easily know the exact bytes written without buffering.
	// Return an estimate for progress tracking.
	return uint64(image.AVMLHeaderSize + len(block.Data) + image.AVMLTrailerSize), nil
}
