# GOFVML AVML Rewrite Plan: Executive Summary

## Project Name

GOFVML: Go for Volatile Memory Linux.

## Goal

Build a Go-based Linux volatile memory acquisition tool that preserves AVML's core operational strengths while making the implementation easier to inspect, extend, package, and adapt. The first-class compatibility target is AVML's physical-memory acquisition behavior, LiME-compatible output, and Volatility 3 usability. The major product extension is PID-scoped memory dumping. The implementation must also expose an importable Go library so a larger incident-response framework can call acquisition, conversion, and upload workflows directly.

## Source Project Studied

The reference implementation is `/Users/jp/Documents/repos/avml`, version `0.18.0` in `Cargo.toml`. It is a compact Rust crate with three binaries:

- `avml`: acquire volatile memory to a local image, optionally upload after acquisition.
- `avml-convert`: convert between raw, LiME, and AVML-compressed LiME-like formats.
- `avml-upload`: upload an existing image via HTTP PUT or Azure Blob Storage SAS URL.

AVML's library is organized around these modules:

- `iomem`: parse `/proc/iomem` into physical System RAM ranges.
- `snapshot`: choose memory source and transform physical ranges into readable source offsets.
- `image`: write and convert image blocks in LiME or AVML-compressed format.
- `io`: counting and Snappy writer adapters.
- `disk_usage`: preflight disk usage estimation.
- `upload`: HTTP PUT, Azure Blob Storage upload, and optional progress display.
- `errors`: shared error formatting and conversion.

## AVML's Essential Architecture

AVML is a source-to-image pipeline:

1. Parse physical RAM ranges from `/proc/iomem`.
2. Select a source from `/dev/crash`, `/proc/kcore`, or `/dev/mem`.
3. Translate RAM ranges into read blocks for the chosen source.
4. Write each non-zero block as a 32-byte header plus payload.
5. Optionally Snappy-compress each payload and append its compressed length.
6. Optionally upload the completed local file.

The design is intentionally conservative. AVML does not stream directly to cloud storage, does not load kernel modules, and does not need target-specific build steps.

## Key Compatibility Decisions For GOFVML

GOFVML should keep these AVML-compatible behaviors:

- Read `/proc/iomem` and include top-level `System RAM` ranges only.
- Merge overlapping or adjacent RAM ranges before acquisition.
- Prefer acquisition sources in AVML's order when writing to a regular file: `/dev/crash`, then `/proc/kcore`, then `/dev/mem`.
- Select source more carefully for stdout because failed writes cannot be rewound.
- Open output files with owner-only permissions and avoid following symlinks on Unix.
- Write standard LiME headers for uncompressed output.
- Support AVML compressed output with `AVML` magic, version 2 headers, Snappy-framed payloads, and an 8-byte little-endian compressed-length trailer.
- Skip all-zero blocks.
- Split compressed output blocks at `0x1000 * 0x1000` bytes.
- Keep local acquisition and upload as separate phases in the initial implementation.
- Keep default physical output directly usable with Volatility 3's Linux workflow.

## PID-Scoped Dumping Direction

AVML is a physical-memory imager. PID-scoped dumping is a different acquisition mode and should not be forced into the same abstraction too early.

GOFVML should support two families of acquisition:

- `physical`: whole-machine volatile memory, AVML-compatible.
- `process`: virtual memory ranges belonging to one or more Linux processes.

PID mode should use `/proc/<pid>/maps`, `/proc/<pid>/mem`, and process metadata under `/proc/<pid>`. It should produce a self-describing format that can represent virtual address ranges, gaps, permissions, file mappings, and short reads. For analyst interoperability, it can also offer raw segment export, but raw is not enough as the primary format because it loses essential map metadata.

PID-scoped process dumps are not complete Volatility physical memory images. They are separate virtual-memory artifacts that should be correlated with full-system images when Volatility kernel plugins are required.

## Library Direction

The CLI should be a thin wrapper around public Go packages. Larger IR frameworks should be able to import GOFVML and call physical memory acquisition, PID-scoped process dumping, image conversion, upload operations, metadata/sidecar generation, and format readers/writers directly.

Every long-running API should accept `context.Context`, with the parameter named `context`, not `ctx`. Public examples and implementation guidance should prefer descriptive names over terse variables.

## Recommended Output Formats

Implement these in order:

1. `lime`: AVML-compatible physical memory output.
2. `avml-compressed`: AVML-compatible compressed physical memory output.
3. `gofvml-process-v1`: GOFVML-native process dump container with per-range metadata and payload blocks.
4. `raw`: conversion/interoperability format for simple workflows.

## Recommended Go Package Shape

Proposed package layout:

- `cmd/gofvml`: main acquisition CLI.
- `cmd/gofvml-convert`: conversion CLI.
- `cmd/gofvml-upload`: upload CLI.
- `internal/iomem`: `/proc/iomem` parser and range utilities.
- `internal/source`: memory source detection and read adapters.
- `internal/snapshot`: physical acquisition orchestration.
- `internal/image`: LiME and AVML-compressed encoders/decoders.
- `internal/process`: PID map parsing and process memory acquisition.
- `internal/procfs`: focused procfs readers.
- `internal/diskusage`: disk usage preflight checks.
- `internal/upload`: HTTP PUT and Azure uploader.
- `internal/progress`: optional progress reporting.
- `internal/cli`: shared CLI parsing/validation.

## MVP Boundary

The MVP should be intentionally sharp:

- Physical acquisition from `/dev/crash`, `/proc/kcore`, `/dev/mem`.
- LiME and AVML-compressed write path.
- Convert between raw, LiME, and AVML-compressed.
- PID dump for one PID using `/proc/<pid>/maps` and `/proc/<pid>/mem`.
- PID range filtering by permissions and optional explicit address range.
- Tests ported from AVML for `iomem`, image headers, zero-block behavior, conversion, and block splitting.

Not in the first milestone:

- Live cloud streaming during acquisition.
- Kernel module loading.
- Windows/macOS support.
- Full Volatility plugin compatibility for process dumps.
- Memory writes or process mutation.
- eBPF, ptrace-heavy features, or kernel agent mode.

## Principal Risks

- Linux lockdown and kernel hardening can block all physical acquisition sources.
- `/proc/<pid>/mem` access depends on ptrace/Yama policy, credentials, dumpability, namespaces, and process lifetime.
- PID dumps can be inconsistent if a process mutates mappings during acquisition.
- AVML compressed format is simple but non-standard; compatibility tests must be exact.
- `/proc/kcore` virtual-to-physical translation is subtle and must be regression tested against real hosts.

## Documentation Set

This planning corpus is split into:

- `01-avml-component-inventory.md`: every component in AVML and its responsibilities.
- `02-physical-acquisition-architecture.md`: source selection, range mapping, and snapshot flow.
- `03-image-formats-and-conversion.md`: LiME, AVML compressed, raw conversion, and exact encoding rules.
- `04-pid-scoped-memory-dumping.md`: process dumping design and Linux constraints.
- `05-go-architecture.md`: proposed GOFVML packages, interfaces, and CLI.
- `06-implementation-roadmap.md`: milestone-by-milestone build plan.
- `07-testing-and-validation.md`: unit, integration, privilege, and compatibility testing plan.
- `08-open-questions-and-assumptions.md`: decisions to validate before and during implementation.
- `09-source-to-go-mapping.md`: direct mapping from AVML source files to GOFVML packages and backlog items.
- `10-volatility3-compatibility.md`: Volatility 3 workflow, LiME expectations, symbol table needs, and external validation plan.
- `11-library-api-and-interfaces.md`: importable Go API design for IR framework integration.
- `12-style-interface-and-test-standards.md`: naming, interface, and test standards including the no-`ctx` convention.
