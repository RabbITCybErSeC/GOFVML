# Importable Go Library And Interface Design

## Library Goal

GOFVML must be usable in two modes:

- A standalone command-line acquisition tool.
- An importable Go library for larger incident-response frameworks.

The CLI should be a thin adapter over the library. Core acquisition, image encoding, PID dumping, conversion, and upload behavior should be callable from Go code without shelling out.

## API Design Principles

Follow these principles:

- Contract first: define stable interfaces before wiring CLI details.
- Make valid usage obvious and invalid usage hard.
- Keep implementation packages internal until the API settles.
- Avoid leaking device-specific details into high-level callers.
- Return structured results and typed errors instead of printing.
- Accept `context.Context` for cancellation, but name the parameter `context`, not `ctx`.
- Use descriptive variable names in public and internal code. Avoid short names such as `ctx`, `cfg`, `src`, `dst`, `r`, `w`, `b`, and `err` where clarity suffers. The ordinary `err` exception is acceptable in tiny local checks, but public examples should use descriptive names.

## Proposed Public Packages

Start with a small public surface:

```text
pkg/gofvml
pkg/gofvml/physical
pkg/gofvml/process
pkg/gofvml/image
pkg/gofvml/convert
pkg/gofvml/upload
```

Everything else remains under `internal/`.

If this feels too broad during implementation, begin with only:

```text
pkg/gofvml
pkg/gofvml/image
```

and keep physical/process/upload in `internal` until the first API review. The key is that the CLI must still call library-style functions, not contain business logic.

## Root Package

Package:

```text
pkg/gofvml
```

Responsibilities:

- Version metadata.
- Shared option and result types if they are truly cross-domain.
- Error classification helpers.

Example:

```go
package gofvml

type VersionInfo struct {
    Version   string
    Commit    string
    BuildDate string
}

type ProgressEvent struct {
    Operation      string
    BytesCompleted uint64
    BytesTotal     uint64
    Message        string
}

type ProgressFunc func(event ProgressEvent)
```

Avoid making the root package a junk drawer. Domain-specific types belong in domain packages.

## Physical Acquisition API

Package:

```text
pkg/gofvml/physical
```

Primary API:

```go
type Format string

const (
    FormatLime           Format = "lime"
    FormatAVMLCompressed Format = "avml-compressed"
)

type SourceMode string

const (
    SourceAuto      SourceMode = "auto"
    SourceDevCrash  SourceMode = "/dev/crash"
    SourceProcKcore SourceMode = "/proc/kcore"
    SourceDevMem    SourceMode = "/dev/mem"
)

type Options struct {
    OutputPath                 string
    Format                     Format
    Source                     SourceMode
    RawSourcePath              string
    MaxDiskUsageBytes          uint64
    MaxDiskUsagePercentage     float64
    WriteMetadataSidecar       bool
    Progress                   gofvml.ProgressFunc
}

type Result struct {
    OutputPath          string
    MetadataSidecarPath string
    Format              Format
    Source              SourceMode
    Ranges              []Range
    BytesWritten        uint64
    StartedAt           time.Time
    CompletedAt         time.Time
    Warnings            []Warning
}

func Acquire(context context.Context, options Options) (Result, error)
```

Notes:

- The parameter name should be `context context.Context`, not `ctx`.
- `Acquire` writes to `OutputPath`; a future streaming API can be added without changing this contract.
- `RawSourcePath` is used only when `Source` indicates raw source mode or test mode.
- `Result` returns operational details instead of requiring the caller to parse logs.

## Physical Advanced API

For larger IR frameworks, a file-path-only API is not enough. Provide lower-level composition without exposing all internals:

```go
type MemorySource interface {
    Name() string
    ReadBlock(context context.Context, block Block, writer io.Writer) error
    Close() error
}

type ImageEncoder interface {
    WriteBlock(context context.Context, block Block, source MemorySource) error
    Close() error
}

type Snapshotter struct {
    Source  MemorySource
    Encoder ImageEncoder
    Ranges  []Range
}

func (snapshotter *Snapshotter) Create(context context.Context) (Result, error)
```

This enables an IR framework to:

- Supply its own source.
- Supply its own output writer.
- Intercept progress.
- Test acquisition with synthetic memory.
- Swap encoders without changing source logic.

## Process Acquisition API

Package:

```text
pkg/gofvml/process
```

Primary API:

```go
type Options struct {
    OutputPath          string
    PIDs                []int
    Format              Format
    IncludeUnreadable   bool
    Strict              bool
    MaxBytes            uint64
    Filters             []MappingFilter
    WriteMetadataSidecar bool
    Progress            gofvml.ProgressFunc
}

type Result struct {
    OutputPath      string
    Processes       []ProcessResult
    BytesWritten    uint64
    StartedAt       time.Time
    CompletedAt     time.Time
    Warnings        []Warning
}

func Dump(context context.Context, options Options) (Result, error)
```

Important:

- PID dumping returns partial success details. If one mapping fails and `Strict` is false, the overall result may have warnings but no fatal error.
- Fatal errors should mean the dump could not be meaningfully created.

Mapping API:

```go
type Mapping struct {
    Start       uint64
    End         uint64
    Permissions string
    Offset      uint64
    Device      string
    Inode       uint64
    Path        string
}

type MappingFilter interface {
    Include(mapping Mapping) bool
}
```

## Image API

Package:

```text
pkg/gofvml/image
```

API:

```go
type Header struct {
    Magic   uint32
    Version uint32
    Start   uint64
    End     uint64
}

func ReadHeader(reader io.Reader) (Header, error)
func WriteHeader(writer io.Writer, header Header) error
func NewLimeEncoder(writer io.Writer) *LimeEncoder
func NewAVMLCompressedEncoder(writer io.Writer) *AVMLCompressedEncoder
```

This package should be usable independently for test fixtures and forensic tooling.

## Conversion API

Package:

```text
pkg/gofvml/convert
```

API:

```go
type Format string

type Options struct {
    SourcePath   string
    OutputPath   string
    SourceFormat Format
    OutputFormat Format
    Strict       bool
}

func Convert(context context.Context, options Options) error
```

Future lower-level streaming conversion:

```go
func ConvertStreams(context context.Context, source io.ReadSeeker, output io.WriteSeeker, options StreamOptions) error
```

Add this only when needed. Do not over-promise seek behavior until the implementation proves it.

## Upload API

Package:

```text
pkg/gofvml/upload
```

API:

```go
type HTTPPutOptions struct {
    FilePath string
    URL      string
    Headers  map[string]string
    Progress gofvml.ProgressFunc
}

func HTTPPut(context context.Context, options HTTPPutOptions) error
```

Azure API later:

```go
type AzureBlobOptions struct {
    FilePath             string
    SASURL               string
    BlockSizeMiB         uint64
    BlockConcurrency     int
    Progress             gofvml.ProgressFunc
}

func AzureBlob(context context.Context, options AzureBlobOptions) error
```

## Error Contracts

Define errors that callers can inspect:

```go
type ErrorCode string

const (
    ErrorPermissionDenied      ErrorCode = "PERMISSION_DENIED"
    ErrorSourceUnavailable     ErrorCode = "SOURCE_UNAVAILABLE"
    ErrorLockedDownKcore       ErrorCode = "LOCKED_DOWN_KCORE"
    ErrorDiskEstimateExceeded  ErrorCode = "DISK_ESTIMATE_EXCEEDED"
    ErrorUnsupportedFormat     ErrorCode = "UNSUPPORTED_FORMAT"
    ErrorInvalidImage          ErrorCode = "INVALID_IMAGE"
    ErrorProcessExited         ErrorCode = "PROCESS_EXITED"
    ErrorPartialProcessRead    ErrorCode = "PARTIAL_PROCESS_READ"
)

type Error struct {
    Code    ErrorCode
    Message string
    Cause   error
    Details map[string]string
}

func (errorValue *Error) Error() string
func (errorValue *Error) Unwrap() error
```

Library callers should be able to use:

```go
var gofvmlError *gofvml.Error
if errors.As(operationError, &gofvmlError) && gofvmlError.Code == gofvml.ErrorPermissionDenied {
    // IR framework can prompt for elevated collection.
}
```

## Interface Stability Rules

Once exported:

- Do not rename exported fields.
- Do not change field meanings.
- Add optional fields instead of changing existing fields.
- Keep enum string values stable.
- Avoid exposing slices that the library mutates after return.
- Document whether functions are safe for concurrent use.
- Treat output file format behavior as API.

Internal packages can move freely until a release.

## Library Testing Requirements

Every public API needs contract tests:

- Successful call with fake source.
- Cancellation through `context.Context`.
- Permission/source failure mapped to typed error.
- Progress callback invoked in monotonic byte order where possible.
- Result fields populated.
- No output path deletion on failed upload.
- No CLI-only behavior required for library correctness.

Add example tests for documentation:

```go
func ExampleAcquire() {
    context := context.Background()
    result, acquireError := physical.Acquire(context, physical.Options{
        OutputPath: "host.lime",
        Format:     physical.FormatLime,
        Source:     physical.SourceAuto,
    })
    if acquireError != nil {
        // handle error
        return
    }
    fmt.Println(result.Format)
}
```

Public examples must use descriptive names, even when Go idioms often use short names.

## CLI As Library Consumer

The CLI should:

- Parse flags.
- Validate obvious input.
- Create library options.
- Call library functions.
- Render result or error.

The CLI should not:

- Parse `/proc/iomem`.
- Open `/dev/mem`.
- Write LiME headers.
- Implement process memory reads.
- Contain upload retry logic.

This keeps the bigger IR framework integration honest because the CLI and library share the same path.

## IR Framework Integration Scenarios

The library should support:

- Acquire memory from a remote response agent and stream progress into central telemetry.
- Dump one suspicious PID selected by a detector.
- Convert compressed output before uploading to case storage.
- Attach sidecar metadata to an incident timeline.
- Upload through framework-owned storage clients instead of GOFVML's uploader.
- Run acquisition with a caller-controlled timeout.

Design consequence:

- Every long operation needs `context.Context`.
- Results need machine-readable metadata.
- Functions must not call `os.Exit`.
- Functions must not write to stdout/stderr except through caller-provided hooks.

