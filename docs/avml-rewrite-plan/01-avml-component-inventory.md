# AVML Component Inventory

## Repository Shape

AVML is a small Rust crate with library code under `src/`, binaries under `src/bin/`, regression inputs under `test/`, snapshot test outputs under `src/snapshots/`, and build/test scripts under `eng/`.

Top-level files:

- `Cargo.toml`: crate metadata, feature flags, dependencies, release profile, and `avml-upload` binary feature gate.
- `Cargo.lock`: pinned Rust dependency graph.
- `README.md`: product behavior, supported sources, usage, build, and Azure testing docs.
- `SECURITY.md`: security reporting policy.
- `RELEASE_PROCESS.md`: release process.
- `LICENSE`: MIT license.

## Cargo Features

Feature model from `Cargo.toml`:

- `default = ["put", "blobstore", "native-tls"]`
- `put`: enables HTTP PUT upload through `reqwest`, `url`, `tokio`, and `tokio-util`.
- `blobstore`: enables Azure Blob Storage upload through `azure_core`, `azure_storage_blob`, `url`, `tokio`, and `async-trait`.
- `status`: enables progress bars through `indicatif`.
- `native-tls`: enables vendored native TLS for request stacks.

The Go rewrite should preserve the idea of optional upload features at build or package level, but Go's static binary story is different. Prefer runtime subcommands with optional dependencies only if they do not bloat the main binary unacceptably. If binary size matters, split upload into `gofvml-upload`.

## Binaries

### `src/bin/avml.rs`

Primary acquisition binary.

Responsibilities:

- Parse CLI options with `clap`.
- Determine output format version: version 1 for LiME, version 2 for compressed.
- Parse `/proc/iomem` through `iomem::parse`.
- Construct `Snapshot` with destination, memory ranges, source, disk usage limits, and image version.
- Call `snapshot.create()`.
- If enabled, upload the completed local file by HTTP PUT and/or Azure SAS URL.
- Delete local file only after successful upload and only if `--delete` was passed.

CLI options:

- `--compress`
- `--source`
- `--max-disk-usage`
- `--max-disk-usage-percentage`
- `--url`
- `--delete`
- `--sas-url`
- `--sas-block-size`
- `--sas-block-concurrency`
- `filename`

Important behavior:

- Disk usage percentage accepts only values in `0.01..100.0`.
- Uploads happen after acquisition, not while reading memory.
- Multiple upload options can be used; delete is controlled by whether at least one upload succeeded and `--delete` is set.

### `src/bin/avml-convert.rs`

Image conversion binary.

Responsibilities:

- Convert LiME to AVML-compressed.
- Convert AVML-compressed to LiME.
- Convert LiME or AVML-compressed to sparse-expanded raw.
- Convert raw to LiME or AVML-compressed.

Supported formats:

- `raw`
- `lime`
- `lime_compressed`

Important behavior:

- Conversion loops until source stream position reaches source file length.
- Raw conversion fills gaps with zero bytes based on header start offsets.
- Raw-to-image encodes input in `MAX_BLOCK_SIZE` chunks.
- Zero blocks can disappear during raw-to-image conversion because `image.copy_block` skips all-zero blocks.
- Same-format conversion returns `NoConversionRequired`.

### `src/bin/avml-upload.rs`

Standalone upload utility.

Subcommands:

- `put <filename> <url>`
- `upload-blob <filename> <url> [--sas-block-concurrency] [--sas-block-size]`

Responsibilities:

- Reuse library uploaders.
- Provide upload functionality independent of acquisition.

## Library Root

### `src/lib.rs`

Library surface and lint posture.

Exports:

- `errors::Error`
- `snapshot::{Snapshot, Source}`
- `upload::blobstore::{BlobUploader, DEFAULT_CONCURRENCY}` when `blobstore` is enabled.
- `upload::http::put` when `put` is enabled.
- `ONE_MB`
- `Result<T>`

Internal modules:

- `disk_usage`
- `errors`
- `image`
- `io`
- `iomem`
- `snapshot`
- `upload`

Lint posture:

- Denies undocumented unsafe blocks, unwraps, expects, panics, manual asserts, indexing/slicing, and several style hazards.
- Warns on arithmetic side effects and numeric conversions.

GOFVML should copy the spirit, not the exact mechanics: narrow unsafe/syscall wrappers, explicit error returns, no silent panics, and tests around arithmetic boundaries.

## Physical Memory Range Parser

### `src/iomem.rs`

Responsibilities:

- Read `/proc/iomem`.
- Select top-level lines ending in ` : System RAM`.
- Skip indented child ranges.
- Parse hexadecimal start/end addresses.
- Detect `00000000-00000000 : System RAM` as permission failure, returning `PermissionDenied`.
- Merge overlapping or adjacent ranges.
- Split ranges for tests and utility behavior.

Important details:

- AVML uses exclusive `Range<u64>` in memory but parses Linux's inclusive textual end as the same numeric end without adding one.
- Tests encode current expected behavior. The code even notes that the range model is effectively inclusive in some places.
- `merge_ranges` sorts by `start` and merges while `range.end >= next.start`.
- `split_ranges` divides large ranges into max-sized ranges without emitting empty ranges.

GOFVML recommendation:

- Define an explicit `Range` type with `Start` inclusive and `End` exclusive.
- While porting, decide whether to preserve AVML's exact end-address behavior for compatibility or fix it with a tested compatibility note. For acquisition compatibility, preserve AVML behavior first.
- Keep parser tests using AVML's `test/iomem*.txt` fixtures.

## Snapshot Orchestration

### `src/snapshot.rs`

Responsibilities:

- Represent memory source variants.
- Pick an acquisition source.
- Preflight `/proc/kcore`.
- Translate `/proc/kcore` ELF PT_LOAD segments into physical blocks.
- Translate `/dev/crash` and `/dev/mem` ranges into physical blocks.
- Enforce disk usage limits before writing blocks.
- Call image writer.

Source variants:

- `/dev/crash`
- `/dev/mem`
- `/proc/kcore`
- raw path

Source selection:

- If user provides `--source`, try only that source.
- If destination is `/dev/stdout`, choose one source before writing:
  - `/proc/kcore` if it passes `is_kcore_ok`.
  - `/dev/crash` if it can be opened.
  - `/dev/mem` if it can be opened.
- Otherwise try:
  - `/dev/crash`
  - `/proc/kcore`
  - `/dev/mem`
- Fail fast if disk usage estimate exceeds configured bounds.
- On total failure, aggregate per-source errors into one final error.

`/proc/kcore` handling:

- `is_kcore_ok` requires metadata length greater than `0x2000` and read-open success.
- Open `/proc/kcore` as an ELF stream.
- Filter PT_LOAD segments.
- Sort segments by virtual address.
- Compute a virtual-to-physical offset using first PT_LOAD virtual address minus first `/proc/iomem` range start.
- Convert PT_LOAD segments to physical `Block{range, offset}`.
- Intersect requested memory ranges with physical segment blocks through `find_kcore_blocks`.

`/dev/crash` and `/dev/mem` handling:

- For `/dev/mem`, each memory range becomes `Block{offset: range.start, range: range}`.
- For `/dev/crash`, each range is truncated down to a page boundary by setting end to `((end >> 12) << 12)`.
- Image reading uses page-aligned reads for canonicalized `/dev/crash`, `/dev/mem`, and `/proc/kcore`.

GOFVML recommendation:

- Keep `Snapshot` as orchestration, not low-level I/O.
- Create separate `Source` implementations for `PhysicalDeviceSource`, `KcoreSource`, and `RawSource`.
- Make source selection testable by injecting filesystem probes.

## Image Writer And Converter

### `src/image.rs`

Responsibilities:

- Define image header format.
- Open source and destination files safely.
- Copy memory blocks from source to destination.
- Skip all-zero blocks.
- Snappy-compress version 2 blocks.
- Convert one block at a time between versions.

Constants:

- `MAX_BLOCK_SIZE = 0x1000 * 0x1000` (16 MiB).
- `PAGE_SIZE = 0x1000`.
- `HEADER_LEN = 32`.
- `LIME_MAGIC = 0x4c69_4d45`.
- `AVML_MAGIC = 0x4c4d_5641`.

Header layout:

- `u32` magic, little endian.
- `u32` version, little endian.
- `u64` range start, little endian.
- `u64` range end minus one, little endian.
- `u64` zero padding, little endian.

Writing:

- Version 1 writes LiME magic and raw payload.
- Version 2 writes AVML magic, Snappy frame payload, then 8-byte little-endian compressed length.
- Blocks larger than `MAX_BLOCK_SIZE` are split only for version 2 before writing.
- Blocks up to `MAX_BLOCK_SIZE` are read fully into memory to check whether all bytes are zero.
- All-zero blocks are skipped entirely, including header.
- Large blocks are streamed without zero-block elision.

Opening behavior:

- Source path is canonicalized.
- Destination is created/truncated with mode `0600` and `O_NOFOLLOW` on Unix.
- `align_src` is true if canonical source path is one of `/dev/crash`, `/dev/mem`, `/proc/kcore`.
- Aligned reads read exact pages into a buffer and write them out.
- Non-aligned reads use bounded stream copy.

Conversion:

- `read_header` validates magic/version/padding.
- Version 1 conversion copies a raw block through `copy_block`.
- Version 2 conversion writes a new header, Snappy-decodes a block, copies the uncompressed bytes, then seeks past the 8-byte compressed-length trailer.

GOFVML recommendation:

- Make the format package pure and unit-testable with `io.Reader`, `io.ReaderAt`, `io.Writer`, and `io.Seeker`.
- Keep file-open safety in a filesystem package.
- Represent compressed trailer explicitly in documentation and tests.

## I/O Adapters

### `src/io/counter.rs`

Simple `Write` adapter that counts successful bytes written.

### `src/io/snappy.rs`

Wraps a Snappy frame encoder in a counting writer. On finalize:

- Flush Snappy encoder.
- Retrieve the counted underlying writer.
- Convert count to `u64`.
- Write count as 8-byte little-endian trailer.

GOFVML recommendation:

- Use `github.com/golang/snappy` for Snappy framing if license/dependency policy permits.
- Wrap writer with a counting writer.
- Ensure trailer count equals the compressed byte count, not uncompressed byte count.

## Disk Usage Preflight

### `src/disk_usage.rs`

Responsibilities:

- Estimate output size from memory ranges.
- Enforce maximum absolute MB usage.
- Enforce maximum disk percentage.
- Query filesystem usage via `statfs64`.

Details:

- Estimate = sum of all range lengths plus roughly 100 KiB padding per range.
- Absolute max converts MiB input to bytes.
- Percentage limit uses current used bytes plus estimated bytes and compares with total bytes times percentage.
- Very large `u64` to `f64` conversion is guarded by an arbitrary high value of 4 exabytes.

GOFVML recommendation:

- Implement with `golang.org/x/sys/unix.Statfs`.
- Keep estimates conservative and documented.
- Treat disk usage check as best effort, not a guarantee.

## Upload Components

### `src/upload/http.rs`

Responsibilities:

- Open file asynchronously.
- Determine size.
- Stream file through request body.
- Update optional progress status.
- Send HTTP PUT.
- Include `x-ms-blob-type: BlockBlob` and `Content-Length`.
- Require success HTTP status.

### `src/upload/blobstore.rs`

Responsibilities:

- Build Azure `BlobClient` from SAS URL.
- Calculate block size and concurrency.
- Wrap file stream in progress-reporting seekable stream.
- Upload with Azure SDK options.

Azure constants:

- Max blocks: 50,000.
- Max block size: 4,000 MiB.
- Max file size: max blocks times max block size.
- Minimum block size: 5 MiB.
- Max/default concurrency: 10.
- Memory threshold: 500 MiB across in-flight blocks.

Concurrency selection:

- User block size is interpreted as MiB and clamped between min and max.
- Auto block size is `ceil(file_size / max_blocks)`, then clamped.
- Concurrency is min of requested/default, memory-limited concurrency, and max concurrency.
- Empty files use default SDK upload options.

### `src/upload/status.rs`

Responsibilities:

- With feature `status`, show progress through `indicatif` only when stdin is a terminal.
- Without status feature, provide no-op implementation with same API.

GOFVML recommendation:

- Keep upload as a separate phase and subcommand in MVP.
- HTTP PUT is simple enough for standard library `net/http`.
- Azure upload is more complex; consider deferring until physical acquisition and conversion are stable, unless cloud upload is a launch requirement.

## Tests And Fixtures

Unit/snapshot tests cover:

- `/proc/iomem` merge/split/parse behavior.
- Disk usage estimation and threshold math.
- Header encoding for LiME and AVML compressed formats.
- All-zero block skipping.
- Snappy trailer length behavior.
- Raw/LiME/compressed round trips.
- Kcore physical block translation.
- Azure block size/concurrency decisions.
- Progress stream reset.

Fixtures:

- `test/iomem.txt`
- `test/iomem-2.txt`
- `test/iomem-3.txt`
- `test/iomem-4.txt`
- `test/iomem-5.txt`
- `src/snapshots/*.snap`

GOFVML should copy fixture intent and expected outputs into Go table tests.

## Engineering Scripts

### `eng/build.sh`

Runs release tests for musl target, docs tests, feature matrix checks, minimal build, default build, upload build, and strips binaries.

### `eng/lint.sh`

Runs rustfmt, clippy with warnings denied, typos, and cargo semver checks.

### `eng/test-azure-image.sh`

Creates an Azure VM, copies `avml`, captures compressed memory image to `/mnt/image.lime`, downloads it, and cleans up the resource group.

### `eng/test-on-azure.sh`

Runs Azure image tests in parallel for SKUs listed in `eng/images.txt`, then runs conversion tests.

### `eng/test-conversion.sh`

For each downloaded image:

- Convert compressed image to uncompressed LiME.
- Recompress it.
- Verify recompressed output is byte-identical.

GOFVML recommendation:

- Build equivalent scripts only after local tests are mature.
- Retain cloud matrix testing because physical acquisition behavior is distro/kernel sensitive.

