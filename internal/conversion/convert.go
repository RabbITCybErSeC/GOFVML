package conversion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/image"
	"github.com/RabbITCybErSeC/gofvml/internal/progress"
)

// Options configures an image conversion operation.
type Options struct {
	// SourceFormat is the input format. If FormatUnknown, auto-detection is used.
	SourceFormat Format
	// TargetFormat is the output format.
	TargetFormat Format
	// ChunkSize is the chunk size for raw input processing (default 1 MiB).
	ChunkSize int64
	// SkipZeroChunks omits all-zero chunks from output when true.
	SkipZeroChunks bool
	// Progress is an optional progress callback.
	Progress progress.Callback
}

// Result holds the outcome of a conversion.
type Result struct {
	// Success is true if conversion completed.
	Success bool
	// BytesRead is the number of bytes read from input.
	BytesRead int64
	// BytesWritten is the number of bytes written to output.
	BytesWritten int64
	// ChunksRead is the number of chunks processed.
	ChunksRead int64
	// ChunksSkipped is the number of zero chunks skipped.
	ChunksSkipped int64
	// SourceFormat is the detected/used source format.
	SourceFormat Format
	// TargetFormat is the target format.
	TargetFormat Format
	// Warnings contains non-fatal issues.
	Warnings []*diagnostic.Diagnostic
}

// AddWarning appends a warning to the result.
func (r *Result) AddWarning(w *diagnostic.Diagnostic) {
	r.Warnings = append(r.Warnings, w)
}

// Convert converts an image from source format to target format.
func Convert(ctx context.Context, input io.Reader, output io.Writer, opts Options) (*Result, error) {
	result := &Result{
		TargetFormat: opts.TargetFormat,
	}

	// Validate target format.
	if opts.TargetFormat == FormatUnknown {
		return nil, diagnostic.FormatError("target format must be specified").
			WithOperation("conversion")
	}

	// Auto-detect source format if needed.
	if opts.SourceFormat == FormatUnknown {
		// We need to peek at the input without consuming it.
		// Use a bufio.Reader or read magic and put it back.
		var magic [4]byte
		n, err := input.Read(magic[:])
		if err != nil {
			return nil, diagnostic.FormatError("unable to read input for format detection").
				WithOperation("conversion").
				WithCause(err)
		}
		if n < 4 {
			return nil, diagnostic.FormatError("input too short for format detection").
				WithOperation("conversion")
		}

		// Put magic back by wrapping in a multi-reader.
		input = io.MultiReader(bytes.NewReader(magic[:n]), input)

		detected, _ := DetectFormat(bytes.NewReader(magic[:n]))
		opts.SourceFormat = detected
	}
	result.SourceFormat = opts.SourceFormat

	// Check for same-format conversion.
	if err := ValidateFormatPair(opts.SourceFormat, opts.TargetFormat); err != nil {
		return nil, diagnostic.FormatError(err.Error()).
			WithOperation("conversion")
	}

	// Set default chunk size.
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 1024 * 1024 // 1 MiB default
	}

	// Set up progress reporter.
	reporter := progress.NewReporter(opts.Progress)
	defer reporter.Close()

	// Perform conversion based on source and target formats.
	var err error
	switch opts.SourceFormat {
	case FormatRaw:
		err = convertRaw(ctx, input, output, opts, result, reporter)
	case FormatLiME:
		err = convertLiME(ctx, input, output, opts, result, reporter)
	case FormatAVML:
		err = convertAVML(ctx, input, output, opts, result, reporter)
	}

	if err != nil {
		return nil, err
	}

	result.Success = true
	return result, nil
}

func convertRaw(ctx context.Context, input io.Reader, output io.Writer, opts Options, result *Result, reporter *progress.Reporter) error {
	// Raw to LiME or AVML: read in chunks, encode each chunk.
	chunkSize := opts.ChunkSize
	currentOffset := uint64(0)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunk := make([]byte, chunkSize)
		n, err := io.ReadFull(input, chunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			return diagnostic.FormatError("read input failed").
				WithOperation("conversion").
				WithCause(err)
		}
		chunk = chunk[:n]

		result.BytesRead += int64(n)
		result.ChunksRead++

		// Check for all-zero chunk.
		if opts.SkipZeroChunks && isAllZeros(chunk) {
			result.ChunksSkipped++
			currentOffset += uint64(n)
			reporter.Report(progress.Event{
				Operation: "conversion",
				Phase:     "processing",
				Current:   uint64(result.BytesRead),
			})
			continue
		}

		// Write chunk in target format.
		var written int64
		var writeErr error
		switch opts.TargetFormat {
		case FormatLiME:
			written, writeErr = writeRawChunkAsLiME(output, currentOffset, chunk)
		case FormatAVML:
			written, writeErr = writeRawChunkAsAVML(output, currentOffset, chunk)
		}

		if writeErr != nil {
			return diagnostic.FormatError("write output failed").
				WithOperation("conversion").
				WithCause(writeErr)
		}

		result.BytesWritten += written
		currentOffset += uint64(n)

		reporter.Report(progress.Event{
			Operation: "conversion",
			Phase:     "writing",
			Current:   uint64(result.BytesRead),
		})

		if err == io.ErrUnexpectedEOF {
			break
		}
	}

	return nil
}

func convertLiME(ctx context.Context, input io.Reader, output io.Writer, opts Options, result *Result, reporter *progress.Reporter) error {
	// LiME to raw or AVML: read LiME blocks, decode, write in target format.
	var rawOffset uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read LiME header.
		header, err := image.DecodeLiMEHeader(input)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return diagnostic.FormatError("read LiME header failed").
				WithOperation("conversion").
				WithCause(err)
		}

		// Read payload.
		payloadLen := header.RangeLen()
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(input, payload); err != nil {
			return diagnostic.FormatError("read LiME payload failed").
				WithOperation("conversion").
				WithCause(err)
		}

		result.BytesRead += int64(image.LiMEHeaderSize) + int64(payloadLen)
		result.ChunksRead++

		// Write in target format.
		var written int64
		var writeErr error
		switch opts.TargetFormat {
		case FormatRaw:
			written, rawOffset, writeErr = writeEncodedBlockAsRaw(output, rawOffset, header.Start, payload)
		case FormatAVML:
			written, writeErr = writeLiMEBlockAsAVML(output, header, payload)
		}

		if writeErr != nil {
			return diagnostic.FormatError("write output failed").
				WithOperation("conversion").
				WithCause(writeErr)
		}

		result.BytesWritten += written

		reporter.Report(progress.Event{
			Operation: "conversion",
			Phase:     "writing",
			Current:   uint64(result.BytesRead),
		})
	}

	return nil
}

func convertAVML(ctx context.Context, input io.Reader, output io.Writer, opts Options, result *Result, reporter *progress.Reporter) error {
	// AVML to raw or LiME: read AVML blocks, decompress, write in target format.
	var rawOffset uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read AVML block (header + payload + trailer).
		header, payload, err := image.DecodeAVMLBlock(input)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return diagnostic.FormatError("read AVML block failed").
				WithOperation("conversion").
				WithCause(err)
		}

		result.BytesRead += int64(image.AVMLHeaderSize) + int64(len(payload)) + int64(image.AVMLTrailerSize)
		result.ChunksRead++

		// Write in target format.
		var written int64
		var writeErr error
		switch opts.TargetFormat {
		case FormatRaw:
			written, rawOffset, writeErr = writeEncodedBlockAsRaw(output, rawOffset, header.Start, payload)
		case FormatLiME:
			written, writeErr = writeAVMLBlockAsLiME(output, header, payload)
		}

		if writeErr != nil {
			return diagnostic.FormatError("write output failed").
				WithOperation("conversion").
				WithCause(writeErr)
		}

		result.BytesWritten += written

		reporter.Report(progress.Event{
			Operation: "conversion",
			Phase:     "writing",
			Current:   uint64(result.BytesRead),
		})
	}

	return nil
}

// Helper functions for writing chunks in different formats.

func writeRawChunkAsLiME(w io.Writer, offset uint64, data []byte) (int64, error) {
	header := image.NewLiMEHeader(offset, offset+uint64(len(data)))
	if err := header.Encode(w); err != nil {
		return 0, err
	}
	if _, err := w.Write(data); err != nil {
		return 0, err
	}
	return int64(image.LiMEHeaderSize + len(data)), nil
}

func writeRawChunkAsAVML(w io.Writer, offset uint64, data []byte) (int64, error) {
	return writeAVMLBlocks(w, offset, data)
}

func writeAVMLBlocks(w io.Writer, offset uint64, data []byte) (int64, error) {
	var total int64
	for _, block := range image.SplitRange(offset, offset+uint64(len(data))) {
		start := block.Start - offset
		end := block.End - offset
		header := image.NewAVMLHeader(block.Start, block.End)
		cw := &countingWriter{w: w}
		if err := image.EncodeAVMLBlock(cw, header, data[start:end]); err != nil {
			return total, err
		}
		total += cw.n
	}
	return total, nil
}

func writeEncodedBlockAsRaw(w io.Writer, currentOffset, start uint64, payload []byte) (int64, uint64, error) {
	if start < currentOffset {
		return 0, currentOffset, fmt.Errorf("encoded block start 0x%x overlaps current raw offset 0x%x", start, currentOffset)
	}

	var written int64
	if gap := start - currentOffset; gap > 0 {
		n, err := io.CopyN(w, zeroReader{}, int64(gap))
		written += n
		if err != nil {
			return written, currentOffset + uint64(written), err
		}
	}
	if _, err := w.Write(payload); err != nil {
		return written, currentOffset + uint64(written), err
	}
	written += int64(len(payload))
	return written, start + uint64(len(payload)), nil
}

func writeLiMEBlockAsAVML(w io.Writer, header *image.LiMEHeader, payload []byte) (int64, error) {
	return writeAVMLBlocks(w, header.Start, payload)
}

func writeAVMLBlockAsLiME(w io.Writer, header *image.AVMLHeader, payload []byte) (int64, error) {
	limeHeader := image.NewLiMEHeader(header.Start, header.ExclusiveEnd())
	if err := limeHeader.Encode(w); err != nil {
		return 0, err
	}
	if _, err := w.Write(payload); err != nil {
		return 0, err
	}
	return int64(image.LiMEHeaderSize + len(payload)), nil
}

func isAllZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
