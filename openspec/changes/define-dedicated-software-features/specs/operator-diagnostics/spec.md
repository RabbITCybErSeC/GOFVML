## ADDED Requirements

### Requirement: Structured diagnostic model
GOFVML SHALL expose structured diagnostics for errors, warnings, partial successes, and operator guidance.

#### Scenario: Diagnostic returned to library caller
- **WHEN** an operation fails or completes with warnings
- **THEN** GOFVML MUST expose machine-readable diagnostic details to library callers.

#### Scenario: Diagnostic rendered by CLI
- **WHEN** the CLI reports a failure or warning
- **THEN** it MUST render the structured diagnostic in operator-readable language without losing the underlying classification.

### Requirement: Privilege and policy diagnostics
GOFVML SHALL diagnose common Linux privilege and hardening failures for physical and process acquisition.

#### Scenario: Physical source permission denied
- **WHEN** a physical source cannot be opened because of permissions or kernel policy
- **THEN** GOFVML MUST identify the source and provide actionable context without claiming acquisition succeeded.

#### Scenario: Process memory permission denied
- **WHEN** process memory cannot be opened or read because of ptrace, Yama, dumpability, namespace, or credential policy
- **THEN** GOFVML MUST report the likely policy category when it can be detected.

### Requirement: Metadata sidecars
GOFVML SHALL support optional metadata sidecars for physical artifacts when metadata cannot be embedded without breaking compatibility.

#### Scenario: Physical metadata sidecar
- **WHEN** physical acquisition writes LiME output and metadata sidecar is enabled
- **THEN** GOFVML MUST write metadata outside the LiME stream so Volatility-compatible output remains clean.

### Requirement: Partial success visibility
GOFVML SHALL distinguish full success, partial success, and fatal failure.

#### Scenario: Process partial success
- **WHEN** a process dump records some mappings and skips others in non-strict mode
- **THEN** GOFVML MUST report partial success with warnings rather than a simple success or fatal failure.

### Requirement: Format limitation guidance
GOFVML SHALL explain format limitations that affect downstream analysis.

#### Scenario: Process dump analysis limitation
- **WHEN** an operator creates a PID-scoped dump
- **THEN** GOFVML MUST provide metadata or diagnostics indicating that the result is a process virtual memory artifact, not a full-system physical image.
