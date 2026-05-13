## ADDED Requirements

### Requirement: Process acquisition mode
GOFVML SHALL provide a process acquisition capability for PID-scoped Linux virtual memory dumping.

#### Scenario: Dump one process
- **WHEN** an operator requests a dump for one accessible PID
- **THEN** GOFVML MUST collect selected readable mappings and write a process memory artifact with process metadata.

#### Scenario: Dump multiple processes
- **WHEN** an operator requests multiple PIDs
- **THEN** GOFVML MUST record per-process results so success, partial success, and failure are distinguishable for each PID.

### Requirement: Process maps metadata
GOFVML SHALL parse `/proc/<pid>/maps` and preserve mapping metadata needed to interpret virtual memory ranges.

#### Scenario: Preserve mapping fields
- **WHEN** GOFVML records a process mapping
- **THEN** the artifact metadata MUST include virtual start, virtual end, permissions, file offset, device, inode, and pathname when present.

### Requirement: Process memory reads
GOFVML SHALL read process memory from Linux process memory interfaces without mutating target memory.

#### Scenario: Read readable mapping
- **WHEN** a selected mapping is readable and accessible
- **THEN** GOFVML MUST read bytes from the mapping and associate the payload with that mapping.

#### Scenario: Mapping read failure
- **WHEN** a selected mapping cannot be fully read and strict mode is disabled
- **THEN** GOFVML MUST preserve the error or short-read event in metadata and continue with remaining mappings.

#### Scenario: Strict mapping failure
- **WHEN** a selected mapping cannot be fully read and strict mode is enabled
- **THEN** GOFVML MUST fail the process acquisition for that PID.

### Requirement: Process dump artifact format
GOFVML SHALL use a metadata-rich process artifact format for PID dumps.

#### Scenario: Native process artifact
- **WHEN** GOFVML writes a process dump
- **THEN** the artifact MUST distinguish virtual addresses from physical addresses and MUST include enough metadata to reconstruct mapping context.

#### Scenario: Not a physical image
- **WHEN** an operator requests a process dump
- **THEN** GOFVML MUST make clear through metadata or diagnostics that the artifact is not a full Volatility physical memory image.

### Requirement: Process acquisition filters
GOFVML SHALL support explicit filtering for process mappings.

#### Scenario: Default readable filter
- **WHEN** no mapping filter is specified
- **THEN** GOFVML MUST select readable mappings and exclude non-readable mappings by default.

#### Scenario: Explicit address range filter
- **WHEN** an operator supplies a virtual address range filter
- **THEN** GOFVML MUST dump only selected mapping bytes that intersect that virtual address range.

### Requirement: Process lifetime races
GOFVML SHALL treat process exit, mapping changes, and partial reads as first-class acquisition outcomes.

#### Scenario: Process exits during acquisition
- **WHEN** the target process exits before acquisition completes
- **THEN** GOFVML MUST report the lifecycle event and any partial data already collected according to strict mode.
