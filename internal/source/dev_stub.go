//go:build !linux

package source

import (
	"context"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// DevMemSource is a stub for non-Linux platforms.
type DevMemSource struct{}

// NewDevMemSource creates a stub DevMemSource.
func NewDevMemSource() *DevMemSource {
	return &DevMemSource{}
}

// Info returns static information.
func (d *DevMemSource) Info() Info {
	return Info{Name: SourceMem, Path: PathDevMem, Priority: 3}
}

// Check always reports unavailable on non-Linux.
func (d *DevMemSource) Check(ctx context.Context) Availability {
	return Availability{
		Available: false,
		Path:      PathDevMem,
		Reason:    "physical memory sources are only supported on Linux",
		Diagnostic: diagnostic.SourceError("unsupported platform").
			WithOperation("physical acquisition").
			WithTarget(PathDevMem).
			WithSuggestion("run on a Linux system"),
	}
}

// Open always fails on non-Linux.
func (d *DevMemSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return nil, diagnostic.SourceError("unsupported platform").
		WithOperation("physical acquisition").
		WithTarget(PathDevMem).
		WithSuggestion("run on a Linux system")
}

// DevCrashSource is a stub for non-Linux platforms.
type DevCrashSource struct{}

// NewDevCrashSource creates a stub DevCrashSource.
func NewDevCrashSource() *DevCrashSource {
	return &DevCrashSource{}
}

// Info returns static information.
func (d *DevCrashSource) Info() Info {
	return Info{Name: SourceCrash, Path: PathDevCrash, Priority: 1}
}

// Check always reports unavailable on non-Linux.
func (d *DevCrashSource) Check(ctx context.Context) Availability {
	return Availability{
		Available: false,
		Path:      PathDevCrash,
		Reason:    "physical memory sources are only supported on Linux",
		Diagnostic: diagnostic.SourceError("unsupported platform").
			WithOperation("physical acquisition").
			WithTarget(PathDevCrash).
			WithSuggestion("run on a Linux system"),
	}
}

// Open always fails on non-Linux.
func (d *DevCrashSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return nil, diagnostic.SourceError("unsupported platform").
		WithOperation("physical acquisition").
		WithTarget(PathDevCrash).
		WithSuggestion("run on a Linux system")
}

// ProcKcoreSource is a stub for non-Linux platforms.
type ProcKcoreSource struct{}

// NewProcKcoreSource creates a stub ProcKcoreSource.
func NewProcKcoreSource() *ProcKcoreSource {
	return &ProcKcoreSource{}
}

// Info returns static information.
func (p *ProcKcoreSource) Info() Info {
	return Info{Name: SourceKcore, Path: PathProcKcore, Priority: 2}
}

// Check always reports unavailable on non-Linux.
func (p *ProcKcoreSource) Check(ctx context.Context) Availability {
	return Availability{
		Available: false,
		Path:      PathProcKcore,
		Reason:    "physical memory sources are only supported on Linux",
		Diagnostic: diagnostic.SourceError("unsupported platform").
			WithOperation("physical acquisition").
			WithTarget(PathProcKcore).
			WithSuggestion("run on a Linux system"),
	}
}

// Open always fails on non-Linux.
func (p *ProcKcoreSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return nil, diagnostic.SourceError("unsupported platform").
		WithOperation("physical acquisition").
		WithTarget(PathProcKcore).
		WithSuggestion("run on a Linux system")
}

var _ Source = (*DevMemSource)(nil)
var _ Source = (*DevCrashSource)(nil)
var _ Source = (*ProcKcoreSource)(nil)
