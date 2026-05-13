## ADDED Requirements

### Requirement: Importable domain APIs
GOFVML SHALL expose importable Go APIs for core acquisition, conversion, upload, progress, results, and diagnostics behavior.

#### Scenario: Library caller acquires physical memory
- **WHEN** a Go caller invokes the physical acquisition API with valid options
- **THEN** GOFVML MUST execute the same domain behavior used by the CLI and return a structured result or error.

#### Scenario: Library caller dumps process memory
- **WHEN** a Go caller invokes the process acquisition API with valid PID options
- **THEN** GOFVML MUST execute the same process acquisition behavior used by the CLI and return per-process results.

### Requirement: CLI uses library behavior
GOFVML CLI commands SHALL be adapters over domain APIs rather than independent implementations.

#### Scenario: CLI and API parity
- **WHEN** a behavior is available through a CLI command and a public or internal domain API
- **THEN** both entry points MUST share the same validation, acquisition, conversion, upload, and diagnostic behavior.

### Requirement: Context-aware long-running operations
GOFVML SHALL accept `context.Context` for long-running library operations.

#### Scenario: Operation cancellation
- **WHEN** a caller cancels the provided context during acquisition, conversion, or upload
- **THEN** GOFVML MUST stop the operation as promptly as practical and return cancellation-aware status.

### Requirement: Structured results
GOFVML SHALL return structured result data from library operations.

#### Scenario: Result includes operational details
- **WHEN** an operation completes successfully or partially
- **THEN** GOFVML MUST return structured details such as output paths, format, bytes written or transferred, timing, warnings, and per-target results where applicable.

### Requirement: Public API clarity
GOFVML SHALL keep public API types explicit and domain-specific.

#### Scenario: Avoid root package junk drawer
- **WHEN** a type only applies to physical acquisition, process acquisition, conversion, or upload
- **THEN** GOFVML MUST define it in the relevant domain package rather than a catch-all root package.
