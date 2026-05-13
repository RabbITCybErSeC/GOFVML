# GOFVML Go Architecture

## Design Principles

GOFVML should be a faithful rewrite where compatibility matters and a Go-native redesign where AVML's implementation is constrained by Rust crate structure.

Principles:

- Keep acquisition orchestration separate from device/file I/O.
- Keep image formats pure and testable.
- Keep procfs parsing deterministic and fixture-driven.
- Prefer typed errors and structured diagnostics.
- Avoid global state.
- Make privilege-sensitive behavior explicit at CLI boundaries.
- Treat PID acquisition as a sibling mode, not an afterthought.

## Proposed Module Layout

```text
cmd/
  gofvml/
  gofvml-convert/
  gofvml-upload/
internal/
  cli/
  diskusage/
  image/
  iomem/
  phys/
  process/
  procfs/
  progress/
  source/
  upload/
  version/
pkg/
  format/
```

Use `internal/` for implementation details, but design the CLI around importable public packages from the beginning. GOFVML is intended to become part of a larger Go-based incident-response framework, so command-line tools must call the same library APIs that external consumers can import.

## Core Types

### Address Ranges

```go
package phys

type Range struct {
    Start uint64
    End   uint64 // exclusive
}

func (r Range) Len() uint64
func (r Range) Empty() bool
func Merge(ranges []Range) []Range
func Split(ranges []Range, max uint64) []Range
```

Compatibility warning:

- AVML's `/proc/iomem` parser currently preserves parsed inclusive end values as `Range.End`.
- GOFVML should start with AVML-compatible fixture expectations, then document any later correction.

### Blocks

```go
type Block struct {
    Offset uint64
    Range  phys.Range
}
```

`Offset` is source address/offset. `Range` is output address range.

### Sources

```go
type Source interface {
    Name() string
    ReadBlock(context context.Context, block Block, writer io.Writer) error
    Close() error
}
```

Alternative lower-level interface:

```go
type ReaderAtSource interface {
    io.ReaderAt
    io.Closer
    Name() string
    RequiresPageReads() bool
}
```

Recommendation:

- Use higher-level `ReadBlock` first. It lets `/dev/crash` and `/proc/kcore` hide alignment details.
- Keep a test fake source that implements deterministic reads.

### Encoders

```go
type Encoder interface {
    WriteBlock(context context.Context, block Block, source Source) error
    Close() error
}
```

Encoders:

- `LiMEEncoder`
- `AVMLCompressedEncoder`
- Later: `ProcessEncoder`

## Package Responsibilities

### `internal/iomem`

Responsibilities:

- Read and parse `/proc/iomem`.
- Return merged physical memory ranges.
- Expose parser that accepts `io.Reader` for tests.
- Expose fixture-compatible parse behavior.

Key functions:

```go
func ParseFile(path string) ([]phys.Range, error)
func Parse(r io.Reader) ([]phys.Range, error)
```

### `internal/source`

Responsibilities:

- Open supported physical memory sources.
- Probe availability.
- Provide source-specific read alignment.
- Provide raw file source for tests.

Source implementations:

- `CrashSource`
- `DevMemSource`
- `KcoreSource`
- `RawSource`

### `internal/phys`

Responsibilities:

- Orchestrate physical acquisition.
- Choose sources.
- Transform ranges into blocks.
- Enforce disk usage preflight.
- Connect source blocks to image encoder.

Potential API:

```go
type SnapshotOptions struct {
    Source                  string
    Format                  image.Format
    MaxDiskUsageBytes       uint64
    MaxDiskUsagePercentage  float64
}

func Create(context context.Context, outputPath string, ranges []phys.Range, options SnapshotOptions) error
```

### `internal/image`

Responsibilities:

- Header encode/decode.
- LiME writer.
- AVML-compressed writer.
- Raw conversion.
- Format conversion.

Subpackages are optional:

```text
internal/image/lime
internal/image/avmlcompressed
internal/image/raw
```

Keep it simple until code size demands splitting.

### `internal/process`

Responsibilities:

- PID acquisition workflow.
- Map filtering.
- Process dump container writer.
- Process dump manifest.
- Access diagnostics.

Types:

```go
type Mapping struct {
    Start, End uint64
    Perms      string
    Offset     uint64
    Dev        string
    Inode      uint64
    Path       string
}

type DumpOptions struct {
    PIDs            []int
    IncludeUnreadable bool
    Strict          bool
    MaxBytes        uint64
    Filters         []Filter
}
```

### `internal/procfs`

Responsibilities:

- Focused readers for `/proc`.
- Parse maps, status, cmdline.
- Read yama `ptrace_scope`.
- Resolve `/proc/<pid>/exe`.

Keep this package boring and fixture-driven.

### `internal/diskusage`

Responsibilities:

- Estimate output size.
- Query filesystem usage.
- Enforce absolute and percentage limits.

Implementation:

- Use `golang.org/x/sys/unix.Statfs`.
- Keep platform-specific files with build tags.

### `internal/upload`

Responsibilities:

- HTTP PUT upload.
- Azure Blob Storage upload if retained.
- Progress hooks.

MVP:

- HTTP PUT first.
- Azure later, because Go Azure SDK integration and block tuning are a separate complexity cluster.

### `internal/cli`

Responsibilities:

- Shared flag parsing helpers.
- Human-readable validation.
- Format/source enum parsing.

Recommended CLI library:

- Standard `flag` is enough but subcommands become verbose.
- `spf13/cobra` is capable but heavy.
- `urfave/cli/v2` is smaller and pleasant.

Conservative recommendation:

- Start with `cobra` only if we want rich subcommands and help immediately.
- Otherwise standard library plus small subcommand switch is easier to audit.

## CLI Proposal

Primary:

```text
gofvml physical [--compress] [--source auto|/dev/crash|/proc/kcore|/dev/mem|raw:<path>] [--max-disk-usage MB] [--max-disk-usage-percentage PCT] <output>
```

Process:

```text
gofvml process --pid PID [--pid PID] [--format gofvml-process-v1|raw-dir] [--range START-END] [--name REGEX] [--strict] <output>
```

Conversion:

```text
gofvml-convert --source-format lime_compressed --format lime <src> <dst>
```

Upload:

```text
gofvml-upload put <filename> <url>
gofvml-upload azure-blob <filename> <sas-url> [--block-size-mib N] [--concurrency N]
```

Compatibility alias:

```text
gofvml [AVML-compatible flags] <output>
```

This alias can map to `physical` for users familiar with AVML.

## Importable Library Proposal

Public packages should be introduced deliberately:

```text
pkg/gofvml
pkg/gofvml/physical
pkg/gofvml/process
pkg/gofvml/image
pkg/gofvml/convert
pkg/gofvml/upload
```

The CLI should parse options and delegate to these packages. It should not contain acquisition logic, image encoding, procfs parsing, or upload implementation details.

All public APIs should use descriptive parameter names:

```go
func Acquire(context context.Context, options physical.Options) (physical.Result, error)
func Dump(context context.Context, options process.Options) (process.Result, error)
func Convert(context context.Context, options convert.Options) error
```

Do not use short names such as `ctx`, `cfg`, `src`, or `dst` in public contracts or examples.

## Error Strategy

Use wrapped errors:

```go
return fmt.Errorf("create snapshot from %s: %w", source.Name(), err)
```

Define sentinel or typed errors where tests need to assert behavior:

```go
var ErrPermissionDenied = errors.New("permission denied")
var ErrLockedDownKcore = errors.New("locked down /proc/kcore")

type DiskEstimateExceeded struct {
    Estimated uint64
    Allowed   uint64
}
```

CLI should print cause chains in verbose mode. Default output should remain concise.

## Dependency Choices

Likely dependencies:

- `golang.org/x/sys/unix`: statfs, open flags, process syscalls.
- `github.com/golang/snappy`: Snappy framing.
- Azure SDK for Go: only when implementing Azure upload.

Avoid unnecessary dependencies for:

- `/proc/iomem` parsing.
- LiME header encoding.
- Basic HTTP PUT.
- Progress display in MVP.

## Build Strategy

Targets:

- Linux amd64 first.
- Linux arm64 later if source behavior is validated.

Static binary:

- Go can build mostly static binaries with `CGO_ENABLED=0` if dependencies permit.
- `x/sys/unix` does not require cgo for many syscalls.
- Azure/TLS dependencies may complicate static size but not usually static linking.

Initial commands:

```text
go test ./...
go build ./cmd/gofvml
go build ./cmd/gofvml-convert
go build ./cmd/gofvml-upload
```

## Internal Boundaries To Protect

Do not let:

- CLI parse `/proc/iomem` directly.
- Source adapters write image headers.
- Image encoders choose acquisition sources.
- PID mode reuse physical range types without explicit virtual address naming.
- Upload code know about memory acquisition internals.
- CLI commands become the only way to use core behavior.

These boundaries will keep the Go rewrite understandable.
