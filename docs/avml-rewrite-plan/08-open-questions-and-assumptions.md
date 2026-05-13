# Open Questions And Assumptions

## Key Assumptions

### Assumption: AVML Compatibility Is Required For Physical Dumps

GOFVML should produce LiME-compatible uncompressed physical images and AVML-compatible compressed images. The default physical output should be usable with Volatility 3.

Validation:

- Build byte-level tests from AVML header vectors.
- Run conversion comparison against AVML-generated fixtures.
- Run Volatility 3 `banners` against a GOFVML LiME image.

### Assumption: PID Dumps Need A Native Container

PID-scoped memory cannot be represented well as LiME without losing virtual mapping metadata.

Validation:

- Implement a small process dump reader.
- Confirm analyst workflows can extract raw mapping payloads and metadata.
- Document that PID process dumps are not full Volatility physical-memory images.

### Assumption: GOFVML Must Be Importable

The CLI is not the only product. GOFVML needs stable Go APIs for larger IR frameworks.

Validation:

- CLI calls public library packages.
- Public example tests compile.
- Typed errors support framework branching.
- Long-running operations accept `context.Context` named `context`.

### Assumption: `/proc/<pid>/mem` Is Acceptable For PID MVP

It is simpler than ptrace-heavy or `process_vm_readv`-first designs.

Validation:

- Test same-UID child process dump.
- Test root dump.
- Document Yama and ptrace limitations.

### Assumption: Upload Can Follow Acquisition

AVML uploads after local acquisition. GOFVML can keep this for MVP.

Validation:

- Confirm target operational workflow has enough disk space or uses max-disk safeguards.
- Revisit streaming upload later if disk footprint is unacceptable.

## Open Questions

### Should GOFVML Preserve AVML's `/proc/iomem` End Semantics Exactly?

AVML parses inclusive textual end addresses into a range end field without adding one, then later treats some ends as inclusive. Fixing this would be more internally correct but could change output by one byte per range.

Recommendation:

- Preserve AVML behavior for v1 compatibility.
- Document it as compatibility behavior.
- Consider `--strict-ranges` or a v2 internal correction only after testing.

### Should Compressed Output Be Byte-Identical To AVML?

Snappy implementations may emit different compressed bytes while decoding to the same payload.

Recommendation:

- Require identical block boundaries, headers, and trailer semantics.
- Require decoded payload equality.
- Treat compressed byte identity as best effort, not guaranteed, unless the same Snappy framing implementation produces stable identical bytes.

### Should PID Dump Stop The Target Process?

Stopping improves consistency but increases intrusiveness and permission requirements.

Recommendation:

- Default to no stop.
- Add `--suspend` later with explicit warning and tests.

### Should PID Dumps Include File-Backed Mappings?

File-backed mappings can often be reconstructed from disk, but memory may contain private modifications and loaded code pages are useful.

Recommendation:

- Include all readable mappings by default.
- Provide filters for users who want smaller dumps.

### Should GOFVML Support Azure Blob Upload In MVP?

AVML supports Azure SAS upload with careful block sizing. Recreating it is straightforward but not core to acquisition correctness.

Recommendation:

- Implement HTTP PUT first.
- Add Azure after acquisition, conversion, and PID mode are stable unless Azure is a launch requirement.

### Should GOFVML Have One Binary Or Three?

AVML has three binaries. A single Go binary with subcommands is convenient; separate binaries preserve AVML's operational model.

Recommendation:

- Build one primary `gofvml` with subcommands.
- Also provide `gofvml-convert` and `gofvml-upload` wrappers or separate commands for AVML familiarity.

## Not Doing Initially

- Kernel module deployment: AVML intentionally avoids it, and GOFVML should too.
- Memory modification: acquisition only.
- Live cloud streaming: useful later, but complicates retries and partial image validity.
- Perfect process consistency: impossible without deeper process control; MVP records best-effort state.
- Full forensic framework integration: define stable output first, adapters later.
- Cross-platform support: Linux first.
- GUI/TUI: command-line incident response workflow first.

## Decisions Needed Before Coding

1. Module path for `go.mod`.
2. Whether to copy AVML fixtures into this repo now.
3. Whether Azure upload is MVP or post-MVP.
4. Whether default CLI should be AVML-compatible flags at root or subcommand-first.
5. Process dump container final binary layout.
6. Whether PID mode should support multiple PIDs in first release.
7. Which public packages are exported in the first release versus kept internal.
8. Whether Volatility 3 smoke tests should run in CI or only manually.

## Recommended Immediate Next Decision

Start with parser and image-format compatibility. That gives GOFVML a stable spine:

- `/proc/iomem` parser.
- range utilities.
- LiME header encoder/decoder.
- AVML-compressed block writer.
- conversion tests.

Once those are right, physical source adapters and PID dumping can be developed independently against known-good encoders.
