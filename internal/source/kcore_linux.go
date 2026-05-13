//go:build linux

package source

import (
	"context"
	"debug/elf"
	"fmt"
	"os"
	"sort"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
	"github.com/RabbITCybErSeC/gofvml/internal/phys"
)

// ProcKcoreSource reads physical memory from /proc/kcore on Linux.
// /proc/kcore is an ELF file where physical memory ranges are mapped
// to file offsets via ELF program headers.
type ProcKcoreSource struct {
	mappings []kcoreMapping
}

// kcoreMapping maps a physical address range to a file offset in /proc/kcore.
type kcoreMapping struct {
	physRange  phys.Range
	fileOffset uint64
}

// NewProcKcoreSource creates a new ProcKcoreSource.
func NewProcKcoreSource() *ProcKcoreSource {
	return &ProcKcoreSource{}
}

// Info returns static information about the /proc/kcore source.
func (p *ProcKcoreSource) Info() Info {
	return Info{
		Name:     SourceKcore,
		Path:     PathProcKcore,
		Priority: 2,
	}
}

// Check verifies /proc/kcore exists and is readable, and probes the ELF headers.
func (p *ProcKcoreSource) Check(ctx context.Context) Availability {
	info, err := os.Stat(PathProcKcore)
	if err != nil {
		if os.IsNotExist(err) {
			return Availability{
				Available: false,
				Path:      PathProcKcore,
				Reason:    "/proc/kcore does not exist",
				Diagnostic: diagnostic.SourceError("source not found").
					WithOperation("physical acquisition").
					WithTarget(PathProcKcore).
					WithCause(err),
			}
		}
		return Availability{
			Available: false,
			Path:      PathProcKcore,
			Reason:    "unable to access /proc/kcore",
			Diagnostic: diagnostic.SourceError("source access denied").
				WithOperation("physical acquisition").
				WithTarget(PathProcKcore).
				WithCause(err).
				WithSuggestion("run with root privileges or CAP_SYS_ADMIN"),
		}
	}

	if info.Mode()&os.ModeDevice == 0 && info.Mode()&os.ModeCharDevice == 0 {
		// /proc/kcore is typically a regular file.
	}

	// Try opening to verify readability and ELF format.
	file, err := os.OpenFile(PathProcKcore, os.O_RDONLY, 0)
	if err != nil {
		return Availability{
			Available: false,
			Path:      PathProcKcore,
			Reason:    fmt.Sprintf("unable to open /proc/kcore: %v", err),
			Diagnostic: diagnostic.SourceError("source open failed").
				WithOperation("physical acquisition").
				WithTarget(PathProcKcore).
				WithCause(err).
				WithSuggestion("run with root privileges or CAP_SYS_ADMIN"),
		}
	}
	defer file.Close()

	// Verify it's a valid ELF file.
	_, err = elf.NewFile(file)
	if err != nil {
		return Availability{
			Available: false,
			Path:      PathProcKcore,
			Reason:    fmt.Sprintf("/proc/kcore is not a valid ELF file: %v", err),
			Diagnostic: diagnostic.SourceError("source format error").
				WithOperation("physical acquisition").
				WithTarget(PathProcKcore).
				WithCause(err).
				WithSuggestion("kernel may not expose /proc/kcore as ELF"),
		}
	}

	return Availability{
		Available: true,
		Path:      PathProcKcore,
		Reason:    "/proc/kcore is accessible and valid ELF",
	}
}

// Open opens /proc/kcore and builds the physical-to-file offset mapping.
func (p *ProcKcoreSource) Open(ctx context.Context) (Reader, *diagnostic.Diagnostic) {
	file, err := os.OpenFile(PathProcKcore, os.O_RDONLY, 0)
	if err != nil {
		return nil, diagnostic.SourceError("failed to open /proc/kcore").
			WithOperation("physical acquisition").
			WithTarget(PathProcKcore).
			WithCause(err).
			WithSuggestion("run with root privileges or CAP_SYS_ADMIN")
	}

	elfFile, err := elf.NewFile(file)
	if err != nil {
		file.Close()
		return nil, diagnostic.SourceError("failed to parse /proc/kcore ELF").
			WithOperation("physical acquisition").
			WithTarget(PathProcKcore).
			WithCause(err).
			WithSuggestion("kernel may not expose /proc/kcore as ELF")
	}

	// Build physical-to-file offset mappings from program headers.
	var mappings []kcoreMapping
	for _, prog := range elfFile.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		mappings = append(mappings, kcoreMapping{
			physRange: phys.Range{
				Start: prog.Vaddr,
				End:   prog.Vaddr + prog.Memsz,
			},
			fileOffset: prog.Off,
		})
	}

	// Sort by physical address for binary search.
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].physRange.Start < mappings[j].physRange.Start
	})

	p.mappings = mappings

	return &kcoreReader{
		file:     file,
		mappings: mappings,
	}, nil
}

// kcoreReader implements Reader for /proc/kcore with physical-to-file offset translation.
type kcoreReader struct {
	file     *os.File
	mappings []kcoreMapping
}

// ReadAt reads bytes from /proc/kcore at the given physical offset.
func (k *kcoreReader) ReadAt(p []byte, off uint64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Find the mapping that contains the starting physical address.
	mappingIdx := -1
	for i, m := range k.mappings {
		if m.physRange.Contains(off) {
			mappingIdx = i
			break
		}
		if m.physRange.Start > off {
			break
		}
	}

	if mappingIdx < 0 {
		return 0, NewNotMappedError(off, PathProcKcore)
	}

	mapping := k.mappings[mappingIdx]
	offsetInMapping := off - mapping.physRange.Start
	fileOffset := mapping.fileOffset + offsetInMapping

	return k.file.ReadAt(p, int64(fileOffset))
}

// Close closes the underlying file.
func (k *kcoreReader) Close() error {
	return k.file.Close()
}

// Ensure ProcKcoreSource implements Source.
var _ Source = (*ProcKcoreSource)(nil)

// Ensure kcoreReader implements Reader.
var _ Reader = (*kcoreReader)(nil)
