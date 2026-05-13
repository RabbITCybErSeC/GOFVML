## 1. OpenSpec Baseline

Issue: TBD
Spec refs: `proposal.md`, `design.md`, `specs/**/*.md`, `openspec/config.yaml`

- [ ] 1.1 Review `proposal.md`, `design.md`, and all capability specs for terminology drift against `docs/avml-rewrite-plan`.
- [x] 1.2 Decide first public API scope: use internal domain APIs first, with later public `pkg/gofvml/*` promotion after API review.
- [x] 1.3 Decide first process artifact shape: implement single-file `gofvml-process-v1` EOF-index container first.
- [x] 1.4 Decide upload parity scope: Azure is not required; HTTP PUT is first milestone, with S3 bucket upload as a desirable later extension.
- [x] 1.5 Decide conversion strictness scope: default AVML-compatible behavior first; defer optional strict compressed trailer validation until compatibility tests are green.
- [x] 1.6 Decide Go module path: use `github.com/RabbITCybErSeC/gofvml`.
- [ ] 1.7 Archive the accepted OpenSpec change so capability specs become active under `openspec/specs`.
- [ ] 1.8 Validate the archived active specs with OpenSpec after archive completes.
- [ ] 1.9 Record S3 bucket upload as a backlog candidate or future OpenSpec change, not part of the first implementation milestone.

## 2. Repository Foundation

Issue: TBD
Spec refs: `design.md`, `specs/library-api/spec.md`, `specs/validation-suite/spec.md`

- [x] 2.1 Initialize `go.mod` with module path `github.com/RabbITCybErSeC/gofvml`.
- [x] 2.2 Create command skeletons for `cmd/gofvml`, `cmd/gofvml-convert`, and `cmd/gofvml-upload`.
- [x] 2.3 Create internal package skeletons for `cli`, `diagnostic`, `diskusage`, `image`, `iomem`, `phys`, `process`, `procfs`, `progress`, `source`, `upload`, and `version`.
- [x] 2.4 Create internal library-facing domain package skeletons that can later be promoted to public `pkg/gofvml/*` packages after API review.
- [x] 2.5 Add a minimal `go test ./...` path that passes before privileged functionality exists.
- [x] 2.6 Add fixture directories for iomem, process maps, image headers, conversion samples, raw sources, and upload responses.
- [x] 2.7 Add repository style notes covering descriptive names, `context context.Context`, typed diagnostics, and CLI-as-adapter boundaries.
- [x] 2.8 Add `.gitignore` entries for build outputs, test artifacts, temporary memory images, coverage files, and local validation output.
- [x] 2.9 Update `README.md` with project purpose, Linux privilege warning, module path, initial commands, and OpenSpec-driven implementation status.
- [x] 2.10 Reconcile any repo-level agent or contributor guidance with the new OpenSpec source of truth.

## 3. Shared Types And Diagnostics

Issue: TBD
Spec refs: `specs/operator-diagnostics/spec.md`, `specs/library-api/spec.md`, `specs/validation-suite/spec.md`

- [ ] 3.1 Define shared range/block types with exclusive-end semantics and tests for empty, length, merge, and split behavior.
- [ ] 3.2 Define structured diagnostic categories for source errors, policy errors, parse errors, format errors, partial success, cancellation, and upload errors.
- [ ] 3.3 Define warning/result conventions used by physical, process, conversion, and upload workflows.
- [ ] 3.4 Add CLI diagnostic rendering helpers that preserve machine-readable classifications while producing operator-readable messages.
- [ ] 3.5 Add cancellation test helpers for long-running operations that accept `context.Context`.
- [ ] 3.6 Add diagnostic serialization tests so warnings and errors remain usable by library callers and CLI JSON output later.

## 4. Physical Range Discovery

Issue: TBD
Spec refs: `specs/physical-acquisition/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [x] 4.1 Implement `/proc/iomem` line parsing from an `io.Reader`.
- [x] 4.2 Filter parsed ranges to top-level `System RAM` entries.
- [ ] 4.3 Detect zeroed or permission-hidden `/proc/iomem` output and return a policy-aware diagnostic.
- [x] 4.4 Merge adjacent and overlapping physical ranges before acquisition.
- [x] 4.5 Port AVML iomem fixtures and add tests for malformed lines, empty input, overlap, adjacency, and zeroed-address behavior.
- [x] 4.6 Document the AVML-compatible inclusive/exclusive end handling used by GOFVML range parsing.

## 5. Physical Source Adapters

Issue: TBD
Spec refs: `specs/physical-acquisition/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 5.1 Define the memory source interface used by physical acquisition and image encoders.
- [ ] 5.2 Implement raw/fake source adapters for unprivileged tests.
- [ ] 5.3 Implement `/dev/mem` source opening and page-aware reads with permission diagnostics.
- [ ] 5.4 Implement `/dev/crash` source opening, range handling, and read diagnostics.
- [ ] 5.5 Implement `/proc/kcore` probe and physical-to-file block mapping.
- [ ] 5.6 Add source availability and explicit-source failure tests using fakes or fixture-backed sources.
- [ ] 5.7 Add Linux build tags or platform guards so device-backed sources fail clearly on unsupported platforms.

## 6. Physical Acquisition Workflow

Issue: TBD
Spec refs: `specs/physical-acquisition/spec.md`, `specs/library-api/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 6.1 Implement safe output file creation with `0600` permissions and symlink-safe behavior on Unix.
- [ ] 6.2 Implement disk usage estimate and preflight hooks without requiring privileged devices in tests.
- [ ] 6.3 Implement auto source fallback order for regular output files: `/dev/crash`, `/proc/kcore`, `/dev/mem`.
- [ ] 6.4 Implement explicit source mode that fails without fallback when the requested source is unavailable.
- [ ] 6.5 Implement physical block orchestration from discovered ranges through selected source and encoder.
- [ ] 6.6 Add unprivileged raw-source acquisition tests for success, source failure, output creation failure, and cancellation.
- [ ] 6.7 Add progress events for physical acquisition without coupling the domain workflow to terminal output.

## 7. Image Headers And Encoders

Issue: TBD
Spec refs: `specs/image-conversion/spec.md`, `specs/physical-acquisition/spec.md`, `specs/validation-suite/spec.md`

- [ ] 7.1 Implement LiME 32-byte header encode/decode with exclusive-end to inclusive-end conversion.
- [ ] 7.2 Add LiME byte-vector tests and invalid-header tests for magic, version, padding, overflow, and reversed range.
- [ ] 7.3 Implement AVML-compressed 32-byte header encode/decode with version 2 semantics.
- [ ] 7.4 Implement Snappy-framed compressed payload writing and 8-byte little-endian compressed-length trailers.
- [ ] 7.5 Implement 16 MiB compressed block splitting.
- [ ] 7.6 Implement all-zero block detection and omission for eligible LiME and AVML-compressed blocks.
- [ ] 7.7 Add tests for compressed trailer lengths, block splitting, zero-block omission, and deterministic fake-source reads.
- [ ] 7.8 Add malformed compressed stream tests for truncated Snappy payloads, missing trailers, and impossible block lengths.

## 8. Image Conversion Workflow

Issue: TBD
Spec refs: `specs/image-conversion/spec.md`, `specs/library-api/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 8.1 Implement format detection or explicit format validation for raw, LiME, and AVML-compressed inputs.
- [ ] 8.2 Implement raw-to-LiME conversion with deterministic chunking and zero-chunk skipping.
- [ ] 8.3 Implement raw-to-AVML-compressed conversion with deterministic chunking, compression, trailers, and zero-chunk skipping.
- [ ] 8.4 Implement LiME-to-raw conversion with zero-filled gaps before encoded blocks.
- [ ] 8.5 Implement AVML-compressed-to-raw conversion with decompression and trailer skipping.
- [ ] 8.6 Implement LiME-to-AVML-compressed and AVML-compressed-to-LiME conversion.
- [ ] 8.7 Implement same-format conversion no-op status or error.
- [ ] 8.8 Add sparse round-trip tests, trailing skipped-zero tests, recompression tests, malformed input tests, and cancellation tests.
- [ ] 8.9 Add conversion progress events and byte-count results for large-file workflows.

## 9. Process Metadata And Mapping Selection

Issue: TBD
Spec refs: `specs/process-acquisition/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 9.1 Implement procfs readers for process status, command line, executable path, maps, and Yama ptrace scope.
- [ ] 9.2 Implement `/proc/<pid>/maps` parsing with start, end, permissions, offset, device, inode, and pathname preservation.
- [ ] 9.3 Implement default readable mapping selection.
- [ ] 9.4 Implement mapping filters for permissions, virtual address ranges, pathname matching, and max bytes.
- [ ] 9.5 Add tests for maps fixtures, special mapping names, malformed maps lines, default filtering, range intersections, and byte caps.
- [ ] 9.6 Add diagnostics for missing procfs files, zombie/exited processes, and namespace-related visibility gaps.

## 10. Process Memory Acquisition

Issue: TBD
Spec refs: `specs/process-acquisition/spec.md`, `specs/library-api/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 10.1 Define process acquisition options, per-process result, per-mapping result, and read event structures.
- [ ] 10.2 Implement read-only `/proc/<pid>/mem` opening with privilege and policy diagnostics.
- [ ] 10.3 Implement chunked virtual memory reads for selected mappings.
- [ ] 10.4 Record short reads, read errors, unmapped ranges, process exits, and mapping races as acquisition events.
- [ ] 10.5 Implement non-strict mode that continues after mapping failures and reports partial success.
- [ ] 10.6 Implement strict mode that fails the target PID when selected mappings cannot be fully read.
- [ ] 10.7 Add controlled child-process tests with a known marker and tests for simulated read errors, partial success, strict failure, and cancellation.
- [ ] 10.8 Add process acquisition progress events with per-PID and per-mapping context.

## 11. Process Artifact Format

Issue: TBD
Spec refs: `specs/process-acquisition/spec.md`, `specs/library-api/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 11.1 Define `gofvml-process-v1` file header and version metadata.
- [ ] 11.2 Define process metadata schema including host, kernel, timestamp, PID, command line, executable path, mappings, options, and read events.
- [ ] 11.3 Implement payload block writing with virtual address, mapping index, payload offset/length, compression type, and status.
- [ ] 11.4 Implement artifact finalization using a single-file EOF-index strategy.
- [ ] 11.5 Implement a minimal reader/validator for tests.
- [ ] 11.6 Add tests proving virtual address metadata survives round trip and that process artifacts are not treated as physical LiME images.
- [ ] 11.7 Document `gofvml-process-v1` layout and compatibility limitations for analysts and downstream tooling.

## 12. Library API And CLI Adapters

Issue: TBD
Spec refs: `specs/library-api/spec.md`, `specs/physical-acquisition/spec.md`, `specs/process-acquisition/spec.md`, `specs/image-conversion/spec.md`, `specs/upload-workflows/spec.md`, `specs/operator-diagnostics/spec.md`

- [ ] 12.1 Define physical acquisition API options/result types and connect them to the physical domain workflow.
- [ ] 12.2 Define process acquisition API options/result types and connect them to the process domain workflow.
- [ ] 12.3 Define conversion API options/result types and connect them to conversion workflows.
- [ ] 12.4 Define upload API options/result types and connect them to upload workflows.
- [ ] 12.5 Define shared progress event and callback types without coupling domain logic to terminal output.
- [ ] 12.6 Implement `gofvml physical` as a CLI adapter over the physical acquisition API.
- [ ] 12.7 Implement `gofvml process` as a CLI adapter over the process acquisition API.
- [ ] 12.8 Implement `gofvml-convert` as a CLI adapter over the conversion API.
- [ ] 12.9 Implement `gofvml-upload` as a CLI adapter over the upload API.
- [ ] 12.10 Add API and CLI parity tests for validation, diagnostics, result fields, and cancellation behavior.
- [ ] 12.11 Add CLI help text tests or snapshots for commands, flags, limitations, and examples.
- [ ] 12.12 Add library usage examples that compile as tests once public APIs are promoted.

## 13. Upload Workflows

Issue: TBD
Spec refs: `specs/upload-workflows/spec.md`, `specs/library-api/spec.md`, `specs/operator-diagnostics/spec.md`, `specs/validation-suite/spec.md`

- [ ] 13.1 Implement HTTP PUT upload for existing local artifacts with content length when available.
- [ ] 13.2 Implement upload progress events independent from CLI rendering.
- [ ] 13.3 Implement post-acquisition upload that starts only after acquisition succeeds.
- [ ] 13.4 Implement delete-after-upload only after confirmed upload success.
- [ ] 13.5 Preserve local artifacts after failed upload, interrupted upload, or cancellation.
- [ ] 13.6 Add local HTTP test server coverage for success, rejection, interruption, progress, delete-after-success, and preserve-after-failure.
- [ ] 13.7 Keep upload destination interfaces destination-neutral so S3 bucket upload can be added later without rewriting HTTP PUT behavior.
- [ ] 13.8 Add upload retry policy decision and tests for the first HTTP PUT implementation.
- [ ] 13.9 Document why Azure is out of first scope and how S3 can be introduced later.

## 14. Metadata Sidecars And Operator Guidance

Issue: TBD
Spec refs: `specs/operator-diagnostics/spec.md`, `specs/physical-acquisition/spec.md`, `specs/process-acquisition/spec.md`, `specs/validation-suite/spec.md`

- [ ] 14.1 Define physical metadata sidecar schema for source, ranges, format, timing, warnings, and Volatility-oriented hints.
- [ ] 14.2 Implement optional physical sidecar writing without modifying LiME or AVML-compatible image streams.
- [ ] 14.3 Add operator diagnostics for Linux lockdown, strict devmem, missing `/dev/crash`, inaccessible `/proc/kcore`, Yama ptrace scope, dumpability, namespaces, and target process lifetime.
- [ ] 14.4 Add CLI output tests that distinguish success, partial success, and fatal failure.
- [ ] 14.5 Add documentation for format limitations, privilege requirements, and common remediation paths.
- [ ] 14.6 Add documentation for Volatility 3 validation expectations and when symbol tables are required.
- [ ] 14.7 Add examples for safe local output paths, sidecars, and preserving artifacts after failed upload.

## 15. Validation Suite

Issue: TBD
Spec refs: `specs/validation-suite/spec.md`, all capability specs

- [ ] 15.1 Add unit test coverage for parsers, range utilities, image headers, conversion behavior, mapping filters, diagnostics, and upload decisions.
- [ ] 15.2 Add unprivileged integration tests for raw-source physical acquisition and conversion round trips.
- [ ] 15.3 Add unprivileged integration tests for controlled child-process dumping with known marker validation.
- [ ] 15.4 Add unprivileged integration tests for local HTTP upload workflows.
- [ ] 15.5 Add fuzz or malformed-input tests for `/proc/iomem`, process maps, and image headers.
- [ ] 15.6 Add privileged Linux host validation scripts for `/dev/crash`, `/proc/kcore`, `/dev/mem`, source fallback, and Volatility 3 checks where symbols are available.
- [ ] 15.7 Document the validation matrix, host requirements, expected skipped tests, and release readiness criteria.
- [ ] 15.8 Add `go test -race ./...` guidance and identify which tests are safe to run under the race detector.
- [ ] 15.9 Add large-file or sparse-file validation strategy for conversion without requiring huge committed fixtures.

## 16. Implementation Checkpoints

Issue: TBD
Spec refs: `tasks.md`, all capability specs

- [ ] 16.1 Checkpoint after Tasks 1-4: OpenSpec archived, Go module `github.com/RabbITCybErSeC/gofvml` boots, shared types/diagnostics exist, and parser tests pass.
- [ ] 16.2 Checkpoint after Tasks 5-8: physical acquisition and image conversion work against fake/raw sources with compatibility tests passing.
- [ ] 16.3 Checkpoint after Tasks 9-11: process acquisition can dump a controlled child process and read back artifact metadata.
- [ ] 16.4 Checkpoint after Tasks 12-14: CLIs are adapters over domain APIs, upload works locally, and diagnostics are operator-readable.
- [ ] 16.5 Checkpoint after Task 15: unprivileged validation passes in CI and privileged validation is documented for Linux hosts.

## 17. Implementation Operating Rules

Issue: TBD
Spec refs: `design.md`, `tasks.md`, all capability specs

- [ ] 17.1 Before each implementation tranche, read the relevant active OpenSpec capability and update tasks if scope changed.
- [ ] 17.2 Keep implementation changes incremental and leave `go test ./...` passing after each tranche.
- [ ] 17.3 Update task checkboxes as beads are completed, with no bulk marking at the end.
- [ ] 17.4 Add or update tests in the same tranche as the behavior they cover.
- [ ] 17.5 Run `gofmt` and `go test ./...` before reporting each implementation tranche as complete.
- [ ] 17.6 Track any new architectural decision in OpenSpec or docs before implementing it.
- [ ] 17.7 Preserve unrelated existing workspace changes such as `AGENTS.md` unless the user explicitly asks to modify them.

## 18. Dependency And Security Review

Issue: TBD
Spec refs: `specs/operator-diagnostics/spec.md`, `specs/upload-workflows/spec.md`, `specs/validation-suite/spec.md`, `design.md`

- [ ] 18.1 Keep third-party dependencies minimal and document why each dependency is needed.
- [ ] 18.2 Evaluate Snappy dependency choice for AVML-compressed compatibility before adding it.
- [ ] 18.3 Evaluate any future AWS/S3 dependency separately before adding it to avoid pulling cloud SDK complexity into the HTTP PUT milestone.
- [ ] 18.4 Review all file creation, temporary file, upload deletion, and process memory access paths for evidence-preservation and safety guarantees.
- [ ] 18.5 Add dependency and vulnerability scanning guidance once the module and CI shape exist.

## 19. Packaging And Release Readiness

Issue: TBD
Spec refs: `specs/validation-suite/spec.md`, `specs/operator-diagnostics/spec.md`, `design.md`

- [ ] 19.1 Add build commands for local binaries after command skeletons exist.
- [ ] 19.2 Add static or reproducible Linux build guidance for release candidates.
- [ ] 19.3 Add checksum generation guidance for release artifacts.
- [ ] 19.4 Add installation and usage examples for `gofvml`, `gofvml-convert`, and `gofvml-upload`.
- [ ] 19.5 Add known limitations and threat model notes before the first release candidate.
