# Validation Suite

This document describes the validation strategy for GOFVML.

## Unit Tests

Run unit tests with:

```bash
go test ./...
```

All 22 packages have unit tests covering:
- Parsers: `/proc/iomem`, process maps, LiME/AVML headers
- Range utilities: merge, split, overlap, adjacency
- Image formats: encode/decode, malformed input handling
- Conversion: raw↔LiME↔AVML round trips, zero-block skipping
- Process acquisition: filtering, mapping selection, artifact round-trip
- Upload: HTTP PUT, retry logic, progress events, cancellation
- Diagnostics: structured errors, JSON serialization, collections

## Race Detector

Tests that are safe to run under the race detector:
- `internal/phys` - pure functions, no shared state
- `internal/iomem` - read-only parsers
- `internal/procfs` - read-only parsers
- `internal/image` - header encode/decode (sequential)
- `internal/diagnostic` - no goroutines
- `internal/sidecar` - sequential file writes

Tests that may race (skip with `-race`):
- `internal/progress` - atomic operations should be safe
- `internal/upload` - HTTP client races are external
- `internal/conversion` - sequential but uses io.Copy

Run with race detector:

```bash
go test -race ./internal/phys ./internal/iomem ./internal/procfs ./internal/diagnostic ./internal/sidecar
```

## Integration Tests

### Conversion Round-Trip

`internal/conversion/convert_integration_test.go` validates:
- LiME → raw → LiME preserves content
- AVML → raw → AVML preserves content
- Zero-block skipping reduces output size

### Upload Integration

`internal/upload/upload_test.go` validates:
- HTTP PUT to local test server
- Retry on 5xx errors
- No retry on 4xx errors
- Delete-after-upload on success
- Preserve local file on failure
- Progress events during upload
- Cancellation via context timeout

### Process Acquisition

`internal/process/acquire_test.go` validates:
- Self-process acquisition (Linux only)
- Address filtering
- Progress events
- Invalid PID handling
- No-mappings warning

## Platform-Specific Tests

Tests that require Linux:
- `internal/process` (self-acquisition tests)
- `internal/source` (device source tests)
- `internal/operatordiag` (preflight checks)

These tests are skipped on non-Linux platforms via `runtime.GOOS != "linux"`.

## Privileged Tests

The following require root/CAP_SYS_ADMIN on Linux:
- `/proc/iomem` parsing (zeroed ranges without privileges)
- `/dev/crash`, `/proc/kcore`, `/dev/mem` source access
- Cross-process memory access via `/proc/<pid>/mem`

Unprivileged tests use fake/raw sources and fixture files.

## Fuzz Tests

Fuzz tests are provided for:
- `internal/iomem` - arbitrary /proc/iomem content
- `internal/procfs` - arbitrary /proc/<pid>/maps content

Run fuzz tests:

```bash
go test -fuzz=FuzzParse ./internal/iomem
go test -fuzz=FuzzParseMaps ./internal/procfs
```

## Coverage

Target coverage by package:
- `internal/diagnostic`: >90%
- `internal/phys`: >90%
- `internal/iomem`: >85%
- `internal/procfs`: >85%
- `internal/image`: >85%
- `internal/conversion`: >80%
- `internal/process`: >80%
- `internal/upload`: >80%
- `internal/sidecar`: >80%
- `internal/operatordiag`: >70% (platform-dependent)

## Release Readiness Criteria

Before a release candidate:
1. `go test ./...` passes on Linux and macOS
2. `go test -race` passes on race-safe packages
3. Fuzz tests run for at least 1 minute without crashes
4. CLI binaries build successfully: `go build ./cmd/...`
5. Manual validation on Linux with root privileges

## Known Limitations

- Physical acquisition requires Linux with appropriate privileges
- Process acquisition requires ptrace access or root
- AVML compression depends on Snappy library
- Upload currently supports HTTP PUT only (S3 is a future extension)
