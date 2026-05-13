## ADDED Requirements

### Requirement: Fixture-driven unit tests
GOFVML SHALL include fixture-driven unit tests for parsers, range handling, image headers, conversion behavior, and process maps parsing.

#### Scenario: Parser fixture test
- **WHEN** a parser is implemented for `/proc/iomem`, `/proc/<pid>/maps`, or image headers
- **THEN** tests MUST cover valid fixtures, invalid input, edge cases, and expected error paths.

### Requirement: AVML compatibility validation
GOFVML SHALL validate AVML-compatible behavior with byte-level tests where compatibility is required.

#### Scenario: Header byte compatibility
- **WHEN** GOFVML encodes LiME or AVML-compressed headers
- **THEN** tests MUST compare emitted bytes against known compatible vectors.

#### Scenario: Zero block compatibility
- **WHEN** GOFVML converts or acquires blocks containing only zero bytes
- **THEN** tests MUST verify AVML-compatible skip behavior.

### Requirement: Unprivileged integration tests
GOFVML SHALL provide integration tests that run without root privileges for deterministic workflows.

#### Scenario: Raw source acquisition test
- **WHEN** CI runs without privileged memory devices
- **THEN** GOFVML MUST still be able to test acquisition behavior through raw or fake sources.

#### Scenario: Local upload test
- **WHEN** upload behavior is tested in CI
- **THEN** GOFVML MUST use local test servers or fixtures instead of relying on external network services.

### Requirement: Privileged host validation
GOFVML SHALL document and provide privileged validation workflows for real Linux memory acquisition.

#### Scenario: Real host physical validation
- **WHEN** privileged validation is run on a supported Linux host
- **THEN** GOFVML MUST validate source fallback, physical acquisition, conversion, and downstream Volatility usability where symbols are available.

### Requirement: Process acquisition validation
GOFVML SHALL validate process dumping against controlled target processes.

#### Scenario: Known marker process dump
- **WHEN** GOFVML dumps an accessible child process containing a known marker
- **THEN** tests MUST verify the marker appears in the process dump artifact or exported mapping data.

### Requirement: Error path coverage
GOFVML SHALL test expected failure modes for acquisition, conversion, upload, and diagnostics.

#### Scenario: Expected failure remains informative
- **WHEN** a fixture or integration test triggers a known failure mode
- **THEN** GOFVML MUST assert that the returned diagnostic identifies the operation, cause category, and affected target.
