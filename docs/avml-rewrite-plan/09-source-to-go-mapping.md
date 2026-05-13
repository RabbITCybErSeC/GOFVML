# Source-To-Go Mapping

This file is the bridge from the AVML Rust implementation to the GOFVML implementation backlog. Each AVML component is mapped to a proposed Go package, with notes about what to preserve, what to redesign, and what to test.

## Binary Mapping

| AVML file | Role | GOFVML target | Porting action |
| --- | --- | --- | --- |
| `src/bin/avml.rs` | Main acquisition CLI | `cmd/gofvml` plus `pkg/gofvml/physical` | Preserve flags through compatibility mode; add `physical` and `process` subcommands; CLI delegates to library. |
| `src/bin/avml-convert.rs` | Raw/LiME/compressed conversion | `cmd/gofvml-convert`, `cmd/gofvml convert`, `pkg/gofvml/convert` | Port conversion algorithms after image package is stable. |
| `src/bin/avml-upload.rs` | Standalone upload CLI | `cmd/gofvml-upload`, `cmd/gofvml upload`, `pkg/gofvml/upload` | Implement HTTP PUT first; Azure after core parity. |

## Library Module Mapping

| AVML file | GOFVML package | Preserve | Redesign |
| --- | --- | --- | --- |
| `src/lib.rs` | module root and `internal/*` packages | Small public surface and strict behavior | Go does not need a broad public API initially; keep most code internal. |
| `src/errors.rs` | `internal/apperr` or package-local errors | Cause-chain style and typed failures | Prefer Go error wrapping and typed structs for assertions. |
| `src/iomem.rs` | `internal/iomem`, `internal/phys` | Top-level `System RAM` parsing, merge/split tests, permission detection | Explicit range type and parser from `io.Reader`. |
| `src/snapshot.rs` | `internal/physical`, `internal/source` | Source fallback order, kcore mapper, disk preflight placement | Split source adapters from orchestration. |
| `src/image.rs` | `internal/image` | Header bytes, block rules, zero skip, compressed trailer | Make format code pure over interfaces; isolate file open safety. |
| `src/io/counter.rs` | `internal/image` or `internal/ioutil` | Counting writer behavior | Use small unexported helper. |
| `src/io/snappy.rs` | `internal/image` | Snappy frame plus length trailer | Use Go Snappy dependency with explicit trailer tests. |
| `src/disk_usage.rs` | `internal/diskusage` | Conservative estimate and percentage cap | Use `x/sys/unix.Statfs`; keep math tested. |
| `src/upload/http.rs` | `internal/upload` | PUT body streaming, content length, success-only status | Use standard `net/http`; progress through optional hook. |
| `src/upload/blobstore.rs` | `internal/upload/azure` | Block sizing rules, concurrency caps, progress reset | Defer until required; Azure SDK shape will drive implementation. |
| `src/upload/status.rs` | `internal/progress` | No-op when disabled, terminal-aware progress | Keep optional and non-invasive. |

Public library promotion path:

- `internal/image` becomes or backs `pkg/gofvml/image`.
- `internal/physical` backs `pkg/gofvml/physical`.
- `internal/process` backs `pkg/gofvml/process`.
- `internal/upload` backs `pkg/gofvml/upload`.
- `internal/convert` backs `pkg/gofvml/convert`.

## Test Fixture Mapping

| AVML asset | GOFVML target | Purpose |
| --- | --- | --- |
| `test/iomem.txt` | `internal/iomem/testdata/iomem.txt` | Basic System RAM parse fixture. |
| `test/iomem-2.txt` | `internal/iomem/testdata/iomem-2.txt` | Fragmented ranges. |
| `test/iomem-3.txt` | `internal/iomem/testdata/iomem-3.txt` | Indented child `System RAM` exclusion. |
| `test/iomem-4.txt` | `internal/iomem/testdata/iomem-4.txt` | Large high-memory fixture. |
| `test/iomem-5.txt` | `internal/iomem/testdata/iomem-5.txt` | Many adjacent/fragmented ranges. |
| `src/snapshots/*.snap` | Go table expected values | Reference expected parser/range behavior. |

## CLI Flag Mapping

| AVML flag | GOFVML equivalent | Notes |
| --- | --- | --- |
| `--compress` | `--compress` or `--format avml-compressed` | Preserve short compatibility flag. |
| `--source` | `--source` | Add `auto` and `raw:<path>` conventions. |
| `--max-disk-usage` | `--max-disk-usage` | Same MB semantics. |
| `--max-disk-usage-percentage` | `--max-disk-usage-percentage` | Preserve `0.01..100.0` validation. |
| `--url` | `--url` | HTTP PUT after acquisition. |
| `--delete` | `--delete` | Delete only after successful upload. |
| `--sas-url` | `--sas-url` | Post-MVP unless Azure is required immediately. |
| `--sas-block-size` | `--sas-block-size` | Preserve MiB units. |
| `--sas-block-concurrency` | `--sas-block-concurrency` | Preserve positive integer validation. |
| none | `--pid` | New process dumping mode. |
| none | `--range`, `--name`, `--strict` | New process dumping filters and behavior. |

## Implementation Backlog By Package

### `internal/phys`

Port from:

- `src/iomem.rs` range utilities.
- `src/snapshot.rs` block model.

Backlog:

- `Range` type.
- `Merge`.
- `Split`.
- `Block`.
- Overflow-safe length helpers.

Tests:

- AVML merge snapshots.
- AVML split snapshots.
- Boundary tests.

### `internal/iomem`

Port from:

- `src/iomem.rs`.

Backlog:

- `Parse(io.Reader)`.
- `ParseFile(path string)`.
- permission-denied detection.
- fixture tests.

Tests:

- All AVML `test/iomem*.txt` fixtures.
- invalid lines and parse errors.

### `internal/image`

Port from:

- `src/image.rs`.
- `src/io/counter.rs`.
- `src/io/snappy.rs`.
- conversion logic from `src/bin/avml-convert.rs`.

Backlog:

- `Header`.
- `ReadHeader`.
- `WriteHeader`.
- `LiMEEncoder`.
- `AVMLCompressedEncoder`.
- `Decoder`.
- `Convert`.
- `RawEncode`.
- `RawDecode`.

Tests:

- Header bytes.
- zero block skip.
- compressed trailer.
- sparse raw round trip.
- block split.

### `internal/source`

Port from:

- `src/snapshot.rs`.
- source-opening logic from `src/image.rs`.

Backlog:

- `RawSource`.
- `DeviceSource`.
- `CrashSource`.
- `DevMemSource`.
- `KcoreSource`.
- source probes.
- page-aligned read loop.

Tests:

- fake source reads.
- page read loop.
- source selection with mocked probes.

### `internal/kcore`

Could be subpackage of `source`.

Port from:

- `Snapshot::kcore`.
- `Snapshot::find_kcore_blocks`.

Backlog:

- PT_LOAD parser.
- virtual-to-physical offset calculation.
- block intersection.
- lockdown probe.

Tests:

- AVML translation unit test.
- empty segment list.
- partial intersections.

### `internal/physical`

Port from:

- `Snapshot::create`.
- `Snapshot::phys`.
- disk check integration.

Backlog:

- `CreateSnapshot`.
- explicit source mode.
- auto source mode.
- stdout-safe source mode.
- aggregate errors.

Tests:

- raw source end-to-end.
- source fallback ordering.
- disk estimate failure.

### `internal/process`

New code.

Backlog:

- `Mapping`.
- maps parser.
- metadata collector.
- filter engine.
- process dump writer.
- `/proc/<pid>/mem` reader.
- best-effort error recording.

Tests:

- maps fixtures.
- filter tests.
- child process marker dump.
- process exits mid-dump.

### `internal/diskusage`

Port from:

- `src/disk_usage.rs`.

Backlog:

- `Estimate`.
- `CheckMaxBytes`.
- `CheckMaxPercentage`.
- Unix `Statfs`.

Tests:

- AVML estimate cases.
- synthetic percentage cases.
- statfs smoke test behind build tag if needed.

### `internal/upload`

Port from:

- `src/upload/http.rs`.
- eventually `src/upload/blobstore.rs`.

Backlog:

- HTTP PUT.
- delete-after-success orchestration.
- Azure block sizing calculator.
- Azure upload.

Tests:

- local HTTP server.
- failure preserves file.
- block sizing table tests.

## Critical Compatibility Traps

1. `/proc/iomem` textual end addresses are inclusive, but AVML's in-memory ranges have compatibility quirks. Do not "fix" this silently.
2. AVML compressed blocks end with an 8-byte compressed length trailer. Omitting it breaks conversion.
3. The trailer length is compressed bytes, not raw bytes.
4. Zero blocks are skipped, so decoded raw output may not include a trailing zero-only region.
5. `/dev/crash` range ends are truncated to page boundary.
6. Stdout source selection differs from normal file selection.
7. Output file creation must not follow symlinks.
8. Upload deletion happens only after successful upload.
9. PID dumps need virtual address metadata; LiME alone is the wrong container.
10. Volatility 3 default workflow expects a physical image such as LiME; do not embed GOFVML sidecar metadata inside the LiME stream.
11. Public Go APIs become contracts; avoid short names and unstable exported fields.
