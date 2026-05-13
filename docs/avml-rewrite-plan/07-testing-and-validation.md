# Testing And Validation Plan

## Testing Philosophy

Most GOFVML behavior should be testable without privileges. Privileged tests should be a thin layer on top of deterministic parsers, encoders, converters, and source adapters.

AVML's strongest tests are around:

- `/proc/iomem` fixtures.
- Header byte encoding.
- Zero-block handling.
- Snappy trailer behavior.
- Sparse raw conversion.
- Kcore block translation.
- Azure block sizing.

GOFVML should port those first.

## Unit Tests

### Range Utilities

Test:

- Merge adjacent ranges.
- Merge overlapping ranges.
- Preserve gaps.
- Sort unordered input.
- Split ranges at max size.
- Do not emit empty ranges.
- Saturation/overflow boundaries.

### `/proc/iomem`

Use AVML fixtures:

- `iomem.txt`
- `iomem-2.txt`
- `iomem-3.txt`
- `iomem-4.txt`
- `iomem-5.txt`

Test:

- Top-level filtering.
- Indented child range exclusion.
- Expected parsed ranges.
- `0-0` permission denied.
- Invalid hex.
- Invalid range grammar.

### Image Headers

Test:

- LiME header exact bytes.
- AVML-compressed header exact bytes.
- Padding rejection.
- Unsupported magic/version rejection.
- End overflow rejection.
- End-before-start rejection.

### Encoders

Test:

- LiME writes header plus raw payload.
- AVML-compressed writes header plus Snappy frame plus 8-byte compressed length.
- Compressed length counts compressed bytes only.
- Zero blocks omitted.
- Version 2 splits blocks larger than 16 MiB.
- Version 1 large block behavior matches AVML.

### Converters

Test:

- Raw sparse image to LiME.
- LiME to raw with gaps filled.
- LiME to compressed.
- Compressed to LiME.
- Compressed to raw.
- Trailing zero block is omitted from raw round trip, matching AVML.
- Same-format conversion returns no conversion required.

### Kcore Translation

Port AVML's test:

- Desired ranges: `10..20`, `30..35`, `45..55`.
- Kcore blocks: `10..20`, `25..35`, `40..50`, `50..55`.
- Expected adjusted offsets: `0`, `15`, `25`, `35`.

Add tests:

- Range starts before first kcore block.
- Range spans multiple kcore blocks.
- Range has no backing PT_LOAD segment.
- Empty segment list error.
- Arithmetic underflow/overflow.

### Disk Usage

Test:

- Estimate includes range length plus padding per range.
- Absolute MB cap.
- Percentage cap with synthetic disk usage.
- Disk already over allowed percentage.
- Very large values.

### Process Maps

Test:

- Parse ordinary file-backed mapping.
- Parse anonymous mapping.
- Parse `[heap]`, `[stack]`, `[vdso]`, `[vvar]`.
- Parse deleted file suffix.
- Parse paths with spaces if present.
- Filter readable ranges.
- Filter by permission, name, and address.

## Integration Tests Without Privileges

### Raw Source Acquisition

Build a fake raw memory file:

- Non-zero block.
- Zero gap.
- Non-zero tail.

Acquire through raw source into:

- LiME.
- AVML-compressed.

Then convert back to raw and validate expected sparse behavior.

### Conversion CLI

Run `gofvml-convert` on generated fixtures:

- Raw to LiME.
- LiME to compressed.
- Compressed to LiME.
- LiME to raw.

Validate byte-identical where expected.

### HTTP Upload

Use `httptest.Server`:

- Verify method is PUT.
- Verify content length.
- Verify body bytes.
- Verify optional Azure blob header if implemented.
- Verify non-2xx response returns error.

### PID Dump Child Process

Test process:

- Spawn child process.
- Child allocates known marker string and sleeps.
- Dump child PID.
- Read GOFVML process container.
- Assert marker appears in at least one dumped readable mapping.
- Assert maps metadata is present.

This can usually run unprivileged when parent and child share UID and ptrace policy allows it. If CI blocks it, mark as integration and skip with a clear reason.

## Privileged Tests

Run on controlled Linux VM as root.

Physical source tests:

- Explicit `/dev/crash` if present.
- Explicit `/proc/kcore` if usable.
- Explicit `/dev/mem` if permitted.
- Auto source fallback.
- Lockdown/denial error clarity where applicable.

Validation:

- Output file starts with expected magic.
- Headers parse to plausible physical ranges.
- Conversion to raw succeeds for a bounded test image if feasible.
- Compression conversion round trip succeeds.

PID tests:

- Dump process owned by another UID as root.
- Dump process with many mappings.
- Process exits during dump.
- Mapping is unmapped during dump.

## Compatibility Tests Against AVML

Because GOFVML is a rewrite, compatibility needs explicit checks.

Recommended tests:

1. Generate a deterministic raw source file.
2. Use AVML conversion to produce LiME and compressed outputs.
3. Use GOFVML conversion to produce the same outputs.
4. Compare headers and decoded raw output.
5. For compressed byte identity, be cautious: Snappy library differences may change compressed bytes while preserving decoded data.

What must be byte-identical:

- Headers.
- LiME output for same raw source and zero-block behavior.
- Decoded raw output.

What may not be byte-identical:

- Snappy compressed payloads across library implementations.

If compressed payloads differ:

- Verify they decode to the same bytes.
- Verify trailer length matches actual compressed payload length.
- Verify block boundaries match AVML.

## Volatility 3 Compatibility Tests

Add optional tests that execute Volatility 3 only when explicitly configured. Normal `go test ./...` must not require Python, Volatility, Snappy, or Linux symbol files.

Environment variables:

- `VOL_PY`: path to `vol.py`.
- `VOL_SYMBOLS`: optional path to a Volatility symbols directory.
- `GOFVML_TEST_IMAGE`: optional existing physical image to validate.

Smoke test:

```text
python3 vol.py -f <image> banners
```

Full validation when matching symbols exist:

```text
python3 vol.py -f <image> linux.boottime
python3 vol.py -f <image> linux.pslist
python3 vol.py -f <image> linux.pstree
```

Assertions:

- `banners` exits successfully.
- At least one Linux banner is printed.
- Symbol-dependent plugins are skipped with a clear message when symbols are missing.
- LiME output contains no GOFVML metadata bytes inside the stream.

PID dump note:

- Process dump containers are tested with GOFVML readers and raw mapping extraction, not standard Volatility Linux plugins.
- A future Volatility plugin for `gofvml-process-v1` is a separate compatibility project.

## Fuzz Tests

Targets:

- Image header decoder.
- `/proc/iomem` parser.
- `/proc/<pid>/maps` parser.
- Process dump metadata decoder.

Expected properties:

- No panic.
- No unbounded allocation.
- Structured error on invalid input.
- Decode-encode round trip where input is valid.

## Performance And Stress Tests

Physical acquisition:

- Large sparse raw file.
- Many small ranges.
- One very large range.
- Compressed output memory usage.

PID dump:

- Process with many mappings.
- Large heap mapping.
- Rapidly changing mappings.

Metrics:

- Throughput.
- Peak RSS.
- Output size.
- Error count.

## CI Plan

Default CI:

- `go test ./...`
- `go test -race ./...` if runtime acceptable.
- Fuzz smoke tests with short duration.
- Static analysis with `go vet`.
- Format check with `gofmt`.
- Public API examples compile and use descriptive variable names.

Privileged CI or manual:

- Linux VM matrix.
- Root physical acquisition.
- Cloud image tests.
- Volatility 3 smoke tests.

## Test Data To Import From AVML

Copy or recreate:

- `/Users/jp/Documents/repos/avml/test/iomem.txt`
- `/Users/jp/Documents/repos/avml/test/iomem-2.txt`
- `/Users/jp/Documents/repos/avml/test/iomem-3.txt`
- `/Users/jp/Documents/repos/avml/test/iomem-4.txt`
- `/Users/jp/Documents/repos/avml/test/iomem-5.txt`

Also recreate from AVML tests:

- Header byte vectors.
- Sparse raw fixture generator.
- Kcore block translation case.
- Snappy trailer length case.
