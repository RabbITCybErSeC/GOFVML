//go:build linux

package source

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// DevMemSource reads physical memory from /dev/mem on Linux.
type DevMemSource struct{}

// NewDevMemSource creates a new DevMemSource.
func NewDevMemSource() *DevMemSource {
	return &DevMemSource{}
}

// Info returns static information about the /dev/mem source.
func (d *DevMemSource) Info() Info {
	return Info{
		Name:     SourceMem,
		Path:     PathDevMem,
		Priority: 3,
	}
}

// Check verifies /dev/mem exists and is readable.
func (d *DevMemSource) Check(ctx context.Context) Availability {
	return checkDevice(PathDevMem, SourceMem)
}

// Open opens /dev/mem for reading.
func (d *DevMemSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return openDevice(PathDevMem, SourceMem)
}

// Ensure DevMemSource implements Source.
var _ Source = (*DevMemSource)(nil)

// DevCrashSource reads physical memory from /dev/crash on Linux.
type DevCrashSource struct{}

// NewDevCrashSource creates a new DevCrashSource.
func NewDevCrashSource() *DevCrashSource {
	return &DevCrashSource{}
}

// Info returns static information about the /dev/crash source.
func (d *DevCrashSource) Info() Info {
	return Info{
		Name:     SourceCrash,
		Path:     PathDevCrash,
		Priority: 1,
	}
}

// Check verifies /dev/crash exists and is readable.
func (d *DevCrashSource) Check(ctx context.Context) Availability {
	return checkDevice(PathDevCrash, SourceCrash)
}

// Open opens /dev/crash for reading.
func (d *DevCrashSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	return openDevice(PathDevCrash, SourceCrash)
}

// Ensure DevCrashSource implements Source.
var _ Source = (*DevCrashSource)(nil)

// checkDevice is a helper to check if a device file exists and is readable.
func checkDevice(path, name string) Availability {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Availability{
				Available: false,
				Path:      path,
				Reason:    fmt.Sprintf("%s does not exist", path),
				Diagnostic: diagnostic.SourceError("source device not found").
					WithOperation("physical acquisition").
					WithTarget(path).
					WithCause(err),
			}
		}
		return Availability{
			Available: false,
			Path:      path,
			Reason:    fmt.Sprintf("unable to access %s", path),
			Diagnostic: diagnostic.SourceError("source access denied").
				WithOperation("physical acquisition").
				WithTarget(path).
				WithCause(err).
				WithSuggestion("run with root privileges or CAP_SYS_ADMIN"),
		}
	}

	// Verify it's a device.
	if info.Mode()&os.ModeDevice == 0 && info.Mode()&os.ModeCharDevice == 0 {
		return Availability{
			Available: false,
			Path:      path,
			Reason:    fmt.Sprintf("%s is not a device file", path),
			Diagnostic: diagnostic.SourceError("source is not a device").
				WithOperation("physical acquisition").
				WithTarget(path),
		}
	}

	// Try opening to verify readability.
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		var suggestion string
		if os.IsPermission(err) {
			suggestion = "run with root privileges or CAP_SYS_ADMIN"
		}
		return Availability{
			Available: false,
			Path:      path,
			Reason:    fmt.Sprintf("unable to open %s: %v", path, err),
			Diagnostic: diagnostic.SourceError("source open failed").
				WithOperation("physical acquisition").
				WithTarget(path).
				WithCause(err).
				WithSuggestion(suggestion),
		}
	}
	file.Close()

	return Availability{
		Available: true,
		Path:      path,
		Reason:    fmt.Sprintf("%s is accessible", path),
	}
}

// openDevice is a helper to open a device file for reading.
func openDevice(path, name string) (Reader, *diagnostic.Diagnostic) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		var suggestion string
		if os.IsPermission(err) {
			suggestion = "run with root privileges or CAP_SYS_ADMIN"
		} else if err == syscall.ENODEV {
			suggestion = "kernel may not support this memory source"
		}
		return nil, diagnostic.SourceError("failed to open source device").
			WithOperation("physical acquisition").
			WithTarget(path).
			WithCause(err).
			WithSuggestion(suggestion)
	}

	return NewRawSource(file, path, name), nil
}
