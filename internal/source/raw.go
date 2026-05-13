package source

import (
	"context"
	"os"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// RawSource reads physical memory from a regular file or device.
type RawSource struct {
	file *os.File
	path string
	name string
}

// NewRawSource creates a RawSource from an already-opened file.
func NewRawSource(file *os.File, path, name string) *RawSource {
	return &RawSource{file: file, path: path, name: name}
}

// ReadAt reads bytes from the file at the given offset.
func (r *RawSource) ReadAt(p []byte, off uint64) (int, error) {
	return r.file.ReadAt(p, int64(off))
}

// Close closes the underlying file.
func (r *RawSource) Close() error {
	return r.file.Close()
}

// RawSourceAdapter provides a Source implementation for regular files.
type RawSourceAdapter struct {
	path string
	name string
}

// NewRawSourceAdapter creates a new RawSourceAdapter.
func NewRawSourceAdapter(path, name string) *RawSourceAdapter {
	return &RawSourceAdapter{path: path, name: name}
}

// Info returns static information about the raw source.
func (r *RawSourceAdapter) Info() Info {
	return Info{
		Name:     r.name,
		Path:     r.path,
		Priority: 99, // lowest priority
	}
}

// Check verifies the file exists and is readable.
func (r *RawSourceAdapter) Check(ctx context.Context) Availability {
	info, err := os.Stat(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Availability{
				Available: false,
				Path:      r.path,
				Reason:    "device does not exist",
				Diagnostic: diagnostic.SourceError("source not found").
					WithTarget(r.path).
					WithCause(err),
			}
		}
		return Availability{
			Available: false,
			Path:      r.path,
			Reason:    "unable to access device",
			Diagnostic: diagnostic.SourceError("source access denied").
				WithTarget(r.path).
				WithCause(err).
				WithSuggestion("run with appropriate privileges"),
		}
	}

	if info.Mode()&os.ModeDevice == 0 && info.Mode()&os.ModeCharDevice == 0 {
		// Not a device, but that's okay for raw files and tests.
	}

	// Try opening to verify readability.
	file, err := os.OpenFile(r.path, os.O_RDONLY, 0)
	if err != nil {
		return Availability{
			Available: false,
			Path:      r.path,
			Reason:    "unable to open source for reading",
			Diagnostic: diagnostic.SourceError("source open failed").
				WithTarget(r.path).
				WithCause(err).
				WithSuggestion("run with appropriate privileges"),
		}
	}
	file.Close()

	return Availability{
		Available: true,
		Path:      r.path,
		Reason:    "source accessible",
	}
}

// Open opens the raw source for reading.
func (r *RawSourceAdapter) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	file, err := os.OpenFile(r.path, os.O_RDONLY, 0)
	if err != nil {
		return nil, diagnostic.SourceError("failed to open source").
			WithTarget(r.path).
			WithCause(err).
			WithSuggestion("run with appropriate privileges")
	}

	return NewRawSource(file, r.path, r.name), nil
}

// Ensure RawSource implements Reader.
var _ Reader = (*RawSource)(nil)

// Ensure RawSourceAdapter implements Source.
var _ Source = (*RawSourceAdapter)(nil)
