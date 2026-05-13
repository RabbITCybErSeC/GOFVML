## Why

GOFVML has a strong architectural planning corpus, but the product features are still described mostly as roadmap phases instead of durable OpenSpec capability contracts. Defining dedicated software features now gives implementation work a stable source of truth for behavior, boundaries, tests, and future API compatibility.

## What Changes

- Introduce first-class capability specs for GOFVML's core software surface.
- Define feature boundaries for physical acquisition, PID-scoped process dumping, image conversion, upload, importable Go APIs, operational diagnostics, and validation.
- Preserve AVML compatibility requirements where they are product behavior, not just implementation detail.
- Make CLI features and importable library features share the same behavioral contracts.
- Establish testing and validation expectations per capability before code is written.
- No breaking changes; this repository currently has planning documents and an empty OpenSpec capability set.

## Capabilities

### New Capabilities

- `physical-acquisition`: Whole-machine volatile memory acquisition from Linux physical memory sources with AVML-compatible LiME and compressed output behavior.
- `process-acquisition`: PID-scoped virtual memory dumping using Linux procfs with metadata-rich process dump artifacts.
- `image-conversion`: Conversion between raw, LiME, and AVML-compressed formats with deterministic sparse and compressed behavior.
- `upload-workflows`: Post-acquisition upload workflows for local artifacts, starting with HTTP PUT and preserving local evidence on failure.
- `library-api`: Importable Go APIs that expose acquisition, conversion, upload, progress, result, and diagnostic behavior without shelling out to CLI commands.
- `operator-diagnostics`: Structured errors, warnings, metadata sidecars, and operator-facing guidance for privilege, kernel hardening, access policy, and format limitations.
- `validation-suite`: Unit, fixture, integration, compatibility, and privileged validation expectations for the GOFVML feature set.

### Modified Capabilities

- None.

## Impact

- Affected docs: `docs/avml-rewrite-plan/*` becomes the architectural background, while OpenSpec becomes the requirement source of truth for feature implementation.
- Affected future code: `cmd/gofvml`, `cmd/gofvml-convert`, `cmd/gofvml-upload`, `pkg/gofvml/*`, and internal packages for procfs, physical sources, image formats, upload, diagnostics, and validation helpers.
- Affected APIs: future public Go APIs must follow the capability contracts before being treated as stable.
- Affected testing: each feature requires dedicated test coverage, including fixture-based parser/format tests and privileged validation where host access is required.
