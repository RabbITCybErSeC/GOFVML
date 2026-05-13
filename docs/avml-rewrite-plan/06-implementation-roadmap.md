# Implementation Roadmap

## Phase 0: Repo Foundation

Deliverables:

- `go.mod`
- Basic command directories.
- Initial public package skeleton for importable library APIs.
- Internal package skeleton.
- CI-friendly `go test ./...`.
- README updated with GOFVML purpose and warning that physical memory acquisition needs privileges.

Tasks:

1. Initialize module.
2. Add `internal/phys.Range` and range utilities.
3. Add package-level error conventions.
4. Add fixture directory copied from AVML's `test/iomem*.txt`.
5. Add first unit tests.
6. Add style note requiring descriptive names and `context` instead of `ctx`.

Exit criteria:

- `go test ./...` passes.
- No privileged operations required.

## Phase 1: `/proc/iomem` Parser

Deliverables:

- AVML-compatible parser.
- Merge and split utilities.
- Fixture-driven tests matching AVML expected ranges.

Tasks:

1. Implement top-level `System RAM` filtering.
2. Implement hex range parser.
3. Implement `0-0` permission-denied detection.
4. Implement merge behavior.
5. Port AVML tests.

Exit criteria:

- All AVML iomem fixture expectations pass.
- Edge cases for invalid lines and parse failures are covered.

## Phase 2: Image Format Core

Deliverables:

- LiME header encode/decode.
- AVML-compressed header encode/decode.
- Counting writer.
- Snappy compressed block writer with length trailer.
- Zero-block skip behavior.

Tasks:

1. Implement 32-byte header.
2. Add exact byte-vector tests from AVML.
3. Implement LiME encoder.
4. Implement AVML-compressed encoder.
5. Implement block splitting for compressed output.
6. Implement zero-block omission.

Exit criteria:

- Header tests match AVML byte-for-byte.
- Zero block tests match AVML behavior.
- Snappy trailer length test passes.

## Phase 3: Conversion Tool

Deliverables:

- `gofvml-convert`.
- Raw to LiME.
- Raw to AVML-compressed.
- LiME to raw.
- AVML-compressed to raw.
- LiME to AVML-compressed.
- AVML-compressed to LiME.

Tasks:

1. Implement decoder that walks headers until EOF.
2. Implement gap-filling raw output.
3. Implement raw chunk encoder.
4. Implement same-format no-op error.
5. Port sparse raw round-trip tests.

Exit criteria:

- Sparse raw round trips match AVML tests.
- Recompression tests pass for generated fixtures.

## Phase 4: Physical Source Adapters

Deliverables:

- Raw source.
- `/dev/mem` source.
- `/dev/crash` source.
- `/proc/kcore` source probe and block mapper.

Tasks:

1. Implement safe output open helper with `0600` and no-follow.
2. Implement page-read loop.
3. Implement source probing.
4. Implement `/dev/crash` range truncation.
5. Implement kcore ELF PT_LOAD parser/mapper.
6. Port `find_kcore_blocks` test.

Exit criteria:

- Raw source acquisition works in unprivileged tests.
- Kcore mapping unit tests pass.
- Device sources compile on Linux and return clear errors when unavailable.

## Phase 5: Physical Snapshot CLI

Deliverables:

- `gofvml physical`.
- AVML-compatible shorthand invocation.
- Importable `physical.Acquire` API used by the CLI.
- Disk usage preflight.
- Auto source fallback.
- Optional compression.
- Volatility-oriented LiME default.

Tasks:

1. Wire CLI options to snapshot options.
2. Implement disk estimate and `statfs`.
3. Implement normal file source fallback order.
4. Implement stdout-safe source selection.
5. Add aggregate error reporting.
6. Add optional metadata sidecar for Volatility symbol workflow hints.

Exit criteria:

- Raw source physical acquisition can produce LiME and compressed images.
- CLI help covers sources and limitations.
- Non-root run fails clearly on real physical sources.
- Default LiME output can be validated with Volatility 3 `banners` when a suitable image is available.

## Phase 6: PID Dump MVP

Deliverables:

- `gofvml process --pid`.
- Importable `process.Dump` API used by the CLI.
- `/proc/<pid>/maps` parser.
- `/proc/<pid>/mem` reader.
- GOFVML process dump container v1.
- Manifest/index with metadata and range status.

Tasks:

1. Implement maps parser and tests.
2. Implement process metadata readers.
3. Define container file header and EOF index.
4. Implement mapping filters.
5. Implement chunked `/proc/<pid>/mem` reads.
6. Add child-process integration test with known marker.
7. Add access diagnostics for common policy failures.

Exit criteria:

- Same-UID child process can be dumped and marker is found.
- Failed mappings are recorded without aborting unless `--strict`.
- Container can be read back by a test decoder.
- Documentation clearly says PID dumps are not full Volatility physical images.

## Phase 7: Upload Tools

Deliverables:

- HTTP PUT upload.
- Importable `upload.HTTPPut` API.
- Optional post-acquisition upload.
- Standalone `gofvml-upload put`.
- Azure upload if required for parity.

Tasks:

1. Implement HTTP PUT with content length.
2. Add optional Azure compatibility header for PUT.
3. Wire `--url` and `--delete` to physical command.
4. Add upload tests with local HTTP test server.
5. Implement Azure uploader and AVML-style block sizing only after core acquisition is stable.

Exit criteria:

- HTTP PUT tested locally.
- Delete only happens after successful upload.
- Failed upload preserves local image.

## Phase 8: Real Host Validation

Deliverables:

- Privileged integration test scripts.
- Distro/kernel test matrix.
- Conversion compatibility checks.
- Optional Volatility 3 validation harness.

Tasks:

1. Run on a local Linux VM as root.
2. Validate source fallback behavior with available devices.
3. Validate compressed image conversion.
4. Compare GOFVML LiME structure with AVML on same fixture/raw source.
5. Add Azure or cloud VM matrix later.
6. Run `vol.py -f <image> banners` and selected Linux plugins when symbols are available.

Exit criteria:

- GOFVML can acquire physical memory on at least one supported Linux VM.
- Conversion round trips are stable.
- PID dump works for a controlled process.
- Volatility 3 can stack the default LiME image and detect banners.

## Phase 9: Hardening And Release

Deliverables:

- Static release builds.
- Checksums.
- Operator docs.
- Threat model and limitations.

Tasks:

1. Fuzz parsers: iomem, maps, image headers.
2. Add race and stress tests for process dump.
3. Add large-file tests where feasible.
4. Audit all output file creation.
5. Add clear docs for lockdown, strict devmem, Yama, and privileges.

Exit criteria:

- Release candidate has repeatable builds.
- Known limitations are documented.
- Core tests pass without privileged access.

## Suggested Milestone Order

Build in this order:

1. Parser and format compatibility.
2. Conversion.
3. Raw-source acquisition.
4. Real physical source acquisition.
5. PID dump MVP.
6. Upload parity.
7. Cloud/distro matrix.

Reason:

The image format and conversion layers can be tested deterministically. Once those are correct, physical sources become a source-adapter problem rather than a whole-system debugging problem.
